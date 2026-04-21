class KaraokeRenderer extends LyricsRendererBase {
    constructor(container, stage, config) {
        super(container, stage, config);
        this.interludeEl = null;
        this.params = config?.modeParams?.karaoke || {};
    }

    getDefaultConfig() {
        return {
            wordAnimation: true,
            animationDuration: 0.3,
            currentScale: 1.05
        };
    }

    applyConfig(config) {
        super.applyConfig(config);
        this.params = config?.modeParams?.karaoke || this.getDefaultConfig();
    }

    initStyles() {
        const colors = window.configManager?.getColors() || {};
        const font = window.configManager?.getFont() || {};
        
        document.documentElement.style.setProperty('--fg', colors.text || '#ffffff');
        document.documentElement.style.setProperty('--glow', colors.glow || 'rgba(0,255,255,.9)');
        
        // 应用字体设置
        const fontFamily = font?.family || 'system-ui, -apple-system, Arial';
        const fontSize = font?.size || '2.4rem';
        document.documentElement.style.setProperty('--font-family', fontFamily);
        document.documentElement.style.setProperty('--font-size', fontSize);
        
        this.enableGlow = colors.enableGlow !== false;
        this.glowColor = colors.glow || '#00ffff';
    }

    render(frameData) {
        const motion = window.motion;
        if (!motion?.lrcData?.length) {
            this.renderPlaceholder();
            return;
        }

        const line = motion.lrcData[frameData.currentIndex] || motion.lrcData[0];
        if (!line) return;

        if (this.lastLineIndex !== frameData.currentIndex) {
            this.lastLineIndex = frameData.currentIndex;
            // 优先使用逐字时间戳，如果没有则从文本拆分
            const words = line.words || this.splitTextToWords(line.text);
            this.renderNewLine(line.text, words, line.time);
        }

        this.updateWordProgressByTime(frameData, line);
        this.applyMotionBlur(frameData.velocity);
        this.renderInterlude(frameData);
    }

    // 将文本拆分为单词数组（用于没有逐字时间戳的情况）
    splitTextToWords(text) {
        return text.split(/(?=[^\x00-\xff])/).filter(w => w.trim()).map(w => ({ text: w }));
    }

    renderPlaceholder() {
        this.container.innerHTML = '<span class="shadow">等待播放...</span>';
    }

    renderNoLyrics() {
        this.container.innerHTML = '<span class="shadow" style="color: #ff6b6b;">无法获取歌词</span>';
    }

    // renderNewLine 接收文本和逐字时间戳数组
    renderNewLine(text, words, lineStartTime) {
        this.currentWords = words;
        this.currentLineStartTime = lineStartTime;
        this.container.innerHTML = '';
        this.wordElements = [];

        // 如果有逐字时间戳，为每个字设置时间
        words.forEach((word, i) => {
            const span = document.createElement('span');
            span.className = 'word';
            span.textContent = word.text;
            span.dataset.index = i;
            span.dataset.visible = 'false';
            // 如果有逐字时间戳，存储时间信息
            if (word.time !== undefined) {
                span.dataset.time = word.time;
            }
            this.container.appendChild(span);
            this.wordElements.push(span);
        });

        window.gsap.fromTo(this.container,
            { opacity: 0, y: 20, rotateX: -15 },
            { opacity: 1, y: 0, rotateX: 0, duration: this.params.animationDuration || 0.4, ease: 'power2.out' }
        );
    }

    // 基于时间的进度更新（支持逐字时间戳）
    updateWordProgressByTime(frameData, line) {
        const total = this.wordElements.length;
        if (!total) return;

        // 如果有逐字时间戳，基于当前播放位置计算进度
        if (line.words && line.words.length > 0) {
            const currentTimeMs = frameData.position;
            const words = line.words;

            // 找到当前活跃的字索引
            let activeIndex = -1;
            for (let i = 0; i < words.length; i++) {
                if (currentTimeMs >= words[i].time) {
                    activeIndex = i;
                } else {
                    break;
                }
            }

            this.wordElements.forEach((el, i) => {
                this.updateWordGlowFlow(el, i, activeIndex, currentTimeMs, words);
            });
        } else {
            // 没有逐字时间戳，使用原来的进度百分比方式
            this.updateWordProgress(frameData.lineProgress);
        }
    }

// 更新字的辉光流动效果
    // 逻辑：
    // - 当前字（刚唱到的字）：从初始亮度逐渐变到最亮
    // - 已唱过的字：保持高亮不衰减
    // - 未唱的字：保持基础亮度
    updateWordGlowFlow(el, wordIndex, activeIndex, currentTimeMs, words) {
        if (!this.enableGlow) {
            el.style.textShadow = 'none';
            el.style.opacity = '1';
            return;
        }

        const glowColor = this.glowColor;
        const baseOpacity = 0.7;
        const maxOpacity = 1.0;
        const maxGlowSpread = 20;

        if (wordIndex < activeIndex) {
            // 已唱过的字：保持高亮不衰减
            el.style.opacity = maxOpacity.toFixed(2);
            el.style.textShadow = `0 0 ${maxGlowSpread}px ${glowColor}`;
        } else if (wordIndex === activeIndex) {
            // 当前正唱的字：从暗到亮动画
            if (!el.dataset.animating) {
                el.dataset.animating = 'true';
                gsap.fromTo(el,
                    { opacity: baseOpacity },
                    {
                        opacity: maxOpacity,
                        textShadow: `0 0 ${maxGlowSpread}px ${glowColor}`,
                        duration: 0.3,
                        ease: 'power1.out',
                        onComplete: () => {
                            el.dataset.animating = 'false';
                        }
                    }
                );
            }
        } else {
            // 未唱的字：基础亮度
            el.style.opacity = baseOpacity.toFixed(2);
            el.style.textShadow = `0 0 4px ${glowColor}`;
        }
    }

    updateWordProgress(progress) {
        const total = this.wordElements.length;
        if (!total) return;

        const activeCount = Math.floor(progress * total);
        this.wordElements.forEach((el, i) => {
            const isVisible = i <= activeCount;
            const isCurrent = i === activeCount;

            if (el.dataset.visible !== isVisible.toString()) {
                el.dataset.visible = isVisible.toString();
                if (isVisible && this.params.wordAnimation && this.enableGlow) {
                    this.animateWordGlow(el, isCurrent);
                }
            }
        });
    }

    animateWordGlow(el, isCurrent) {
        const scale = isCurrent ? (this.params.currentScale || 1.05) : 1;
        const glowColor = this.glowColor;

        window.gsap.to(el, {
            textShadow: `0 0 ${isCurrent ? 12 : 6}px ${glowColor}`,
            duration: this.params.animationDuration || 0.3,
            ease: 'power1.out'
        });

        if (isCurrent) {
            window.gsap.fromTo(el,
                { scale: 1, z: 0 },
                { scale: scale, z: 5, duration: 0.15, yoyo: true, repeat: 1 }
            );
        }
    }

    applyMotionBlur(velocity) {
        const blurAmount = Math.min(velocity * 0.1, 3);
        this.container.style.filter = `blur(${blurAmount.toFixed(2)}px)`;
    }

    renderInterlude(frameData) {
        if (frameData.isInterlude && frameData.countdown > 0) {
            if (!this.interludeEl) {
                this.interludeEl = document.createElement('div');
                this.interludeEl.id = 'interlude';
                this.interludeEl.style.cssText = `
                    position: absolute; top: 50%; left: 50%;
                    transform: translate(-50%, -50%);
                    font-size: 4rem; font-weight: bold;
                    color: rgba(255,255,255,0.3);
                    text-shadow: 0 0 20px rgba(0,255,255,0.5);
                `;
                this.stage.appendChild(this.interludeEl);
            }
            this.interludeEl.textContent = frameData.countdown;
            window.gsap.fromTo(this.interludeEl,
                { scale: 1.3, opacity: 0.5 },
                { scale: 1, opacity: 1, duration: 0.5, ease: 'back.out(1.7)' }
            );
        } else if (this.interludeEl) {
            window.gsap.to(this.interludeEl, {
                opacity: 0, duration: 0.3, onComplete: () => {
                    this.interludeEl?.remove();
                    this.interludeEl = null;
                }
            });
        }
    }

    showConnecting(show) {
        if (show) {
            if (!this.connEl) {
                this.connEl = document.createElement('div');
                this.connEl.id = 'connecting';
                this.connEl.style.cssText = `
                    position: fixed; top: 20px; right: 20px;
                    padding: 8px 16px; background: rgba(255,100,0,0.8);
                    color: white; border-radius: 4px; font-size: 14px;
                `;
                document.body.appendChild(this.connEl);
            }
            this.connEl.textContent = '连接中...';
            this.connEl.style.display = 'block';
        } else if (this.connEl) {
            window.gsap.to(this.connEl, {
                opacity: 0, duration: 0.5, onComplete: () => {
                    this.connEl.style.display = 'none';
                }
            });
        }
    }

    clear() {
        super.clear();
        if (this.interludeEl) {
            this.interludeEl.remove();
            this.interludeEl = null;
        }
        if (this.connEl) {
            this.connEl.remove();
            this.connEl = null;
        }
    }

    destroy() {
        super.destroy();
        if (this.interludeEl) {
            this.interludeEl.remove();
            this.interludeEl = null;
        }
        if (this.connEl) {
            this.connEl.remove();
            this.connEl = null;
        }
    }
}

window.KaraokeRenderer = KaraokeRenderer;