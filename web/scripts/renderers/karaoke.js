class KaraokeRenderer extends LyricsRendererBase {
    constructor(container, stage, config) {
        super(container, stage, config);
        this.interludeEl = null;
        this.params = config?.modeParams?.karaoke || {};
        this.wordTimeline = null;
        this.wasPlaying = true;
        this.config = config || {};
        this.textColor = { r: 255, g: 255, b: 255, a: 1 };
        this.lastGlowIndex = -1;
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
        this.config = window.configManager?.get() || {};
        const colors = this.config.colors || {};
        const font = this.config.font || {};

        // 发光配置
        this.glowColor = colors.glow || '#00ffff';
        this.glowIntensity = colors.glowIntensity ?? 1.0;
        this.enableGlow = false; // 禁用发光

        // 解析文字颜色
        this.textColor = this.parseColor(colors.text || '#ffffff');

        document.documentElement.style.setProperty('--fg', colors.text || '#ffffff');

        // 应用字体设置
        const fontFamily = font?.family || 'system-ui, -apple-system, Arial';
        const fontSize = font?.size || '2.4rem';
        document.documentElement.style.setProperty('--font-family', fontFamily);
        document.documentElement.style.setProperty('--font-size', fontSize);

        // 生成样式：每个字使用 gradient 实现变亮效果
        this.generateWordStyles();
    }

    // 解析颜色（支持 #RRGGBB 和 #RRGGBBAA）
    parseColor(color) {
        if (!color) return { r: 255, g: 255, b: 255, a: 1 };

        const hex = color.replace('#', '');
        if (hex.length === 8) {
            return {
                r: parseInt(hex.slice(0, 2), 16),
                g: parseInt(hex.slice(2, 4), 16),
                b: parseInt(hex.slice(4, 6), 16),
                a: parseInt(hex.slice(6, 8), 16) / 255
            };
        }
        return {
            r: parseInt(hex.slice(0, 2), 16),
            g: parseInt(hex.slice(2, 4), 16),
            b: parseInt(hex.slice(4, 6), 16),
            a: 1
        };
    }

    // 调整亮度但保持透明度
    adjustBrightness(color, factor) {
        return {
            r: Math.round(color.r * factor),
            g: Math.round(color.g * factor),
            b: Math.round(color.b * factor),
            a: color.a
        };
    }

    generateWordStyles() {
        // 使用配置中的颜色，暗色为 60% 亮度
        const highlight = this.textColor;
        const dim = this.adjustBrightness(highlight, 0.6);

        const highlightStr = `rgba(${highlight.r}, ${highlight.g}, ${highlight.b}, ${highlight.a})`;
        const dimStr = `rgba(${dim.r}, ${dim.g}, ${dim.b}, ${dim.a})`;

        // 动态插入样式，使用更高优先级
        let styleEl = document.getElementById('karaoke-word-styles');
        if (!styleEl) {
            styleEl = document.createElement('style');
            styleEl.id = 'karaoke-word-styles';
            document.head.appendChild(styleEl);
        }

        // 使用内联样式确保不被 CSS 覆盖
        styleEl.textContent = `
            .karaoke-word {
                display: inline-block;
                background: linear-gradient(to right, ${highlightStr} 50%, ${dimStr} 50%);
                background-size: 200% 100%;
                background-position: 100% 0;
                -webkit-background-clip: text;
                background-clip: text;
                color: transparent;
            }
        `;
    }

    render(frameData) {
        const motion = window.motion;
        if (!motion?.lrcData?.length) {
            this.renderPlaceholder();
            return;
        }

        const line = motion.lrcData[frameData.currentIndex] || motion.lrcData[0];
        if (!line) return;

        const { position, isPlaying, currentIndex } = frameData;

        // 换行检测 - 创建新 timeline
        if (this.lastLineIndex !== currentIndex) {
            this.lastLineIndex = currentIndex;
            const words = line.words || this.splitTextToWords(line.text);
            this.renderNewLine(line.text, words, line.time);

            // 有逐字时间戳时创建 timeline
            if (line.words && line.words.length > 0) {
                this.createWordTimeline(words, line.time);
                
                // 首次加载时根据进度设置状态
                const currentPos = frameData.position;
                const totalDuration = this.wordTimeline.duration() * 1000;
                const targetTime = (currentPos - line.time) / 1000;
                if (targetTime > 0 && targetTime < this.wordTimeline.duration()) {
                    this.wordTimeline.seek(targetTime);
                }
                if (isPlaying) {
                    this.wordTimeline.play();
                }
            } else {
                this.wordTimeline = null;
            }
        }

        // 处理逐字 timeline
        if (this.wordTimeline) {
            const lineStartTime = this.currentLineStartTime;
            const currentPos = frameData.position;

            // 检测进度条拖动到其他行
            const lastWord = this.currentWords[this.currentWords.length - 1];
            const lastWordTime = lastWord?.time || 0;
            if (currentPos < lineStartTime || (lastWordTime > 0 && currentPos > lastWordTime + 500)) {
                this.renderInterlude(frameData);
                return;
            }

            // 暂停状态：用 progress 同步
            if (!isPlaying) {
                const progress = Math.max(0, Math.min(1, (currentPos - lineStartTime) / (this.wordTimeline.duration() * 1000)));
                this.wordTimeline.progress(progress);
                this.wordTimeline.pause();
            } else if (!this.wasPlaying) {
                // 从暂停恢复播放时
                this.wordTimeline.play();
            }
            this.wasPlaying = isPlaying;
        }

        this.renderInterlude(frameData);
    }

    createWordTimeline(words, lineStartTime) {
        const tl = gsap.timeline({ paused: true });

        words.forEach((word, i) => {
            const el = this.wordElements[i];
            // 设置正确的初始背景位置
            el.style.backgroundPositionX = '100%';

            const duration = i === 0
                ? (words[0].time - lineStartTime) / 1000
                : (words[i].time - words[i - 1].time) / 1000;
            const adjustedDuration = Math.max(duration, 0.1);

            // 只做变亮动画，无发光
            tl.to(el, {
                backgroundPositionX: '0%',
                duration: adjustedDuration,
                ease: 'none'
            });
        });

        this.wordTimeline = tl;
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

        // 创建每个字的元素
        words.forEach((word, i) => {
            const span = document.createElement('span');
            span.className = 'karaoke-word';
            span.textContent = word.text;
            // 初始为暗色
            span.style.backgroundPositionX = '100%';
            this.container.appendChild(span);
            this.wordElements.push(span);
        });

        window.gsap.fromTo(this.container,
            { opacity: 0, y: 20, rotateX: -15 },
            { opacity: 1, y: 0, rotateX: 0, duration: this.params.animationDuration || 0.4, ease: 'power2.out' }
        );
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