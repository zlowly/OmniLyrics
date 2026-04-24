class BlurRenderer extends LyricsRendererBase {
    constructor(container, stage, config) {
        super(container, stage, config);
        this.params = config?.modeParams?.blur || {};
        this.lineElements = [];
        
        this.textColor = { r: 255, g: 255, b: 255, a: 1 };
        this.glowRange = 1;
        this.outlineWidth = 1;
        this.outlineColor = { r: 255, g: 255, b: 255, a: 1 };
        this.wasPlaying = true;
        this.wordTimeline = null;
    }

    getDefaultConfig() {
        return {
            visibleLines: 9,
            lineSpacing: 1.5,
            opacityDecay: 0.15,
            blurIncrement: 0.5,
            scaleDecay: 0,
            blurMax: 6,
            wordAnimation: true,
            animationDuration: 0.3,
            currentScale: 1.05
        };
    }

    applyConfig(config) {
        super.applyConfig(config);
        this.params = config?.modeParams?.blur || this.getDefaultConfig();
    }

    initStyles() {
        const colors = window.configManager?.getModeColors('blur') || {};
        const font = window.configManager?.getModeFont('blur') || {};
        
        document.documentElement.style.setProperty('--fg', colors.text || '#ffffff');
        
        const fontFamily = font?.family || 'system-ui, -apple-system, Arial';
        const fontSize = font?.size || '2.4rem';
        const fontWeight = font?.weight || 'normal';
        document.documentElement.style.setProperty('--font-family', fontFamily);
        document.documentElement.style.setProperty('--font-size', fontSize);
        document.documentElement.style.setProperty('--font-weight', fontWeight);
        this.fontWeight = fontWeight;
        
        this.textColor = this.parseColor(colors.text || '#ffffff');
        this.glowRange = colors.glowRange ?? 1;
        this.outlineWidth = colors.outlineWidth ?? 1;
        this.outlineColor = this.parseColor(colors.outlineColor || '#ffffff');
        
        this.generateWordStyles();
    }

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

    adjustBrightness(color, factor) {
        return {
            r: Math.round(color.r * factor),
            g: Math.round(color.g * factor),
            b: Math.round(color.b * factor),
            a: color.a
        };
    }

    generateWordStyles() {
        const highlight = this.textColor;
        const dim = this.adjustBrightness(highlight, 0.6);
        const highlightStr = `rgba(${highlight.r}, ${highlight.g}, ${highlight.b}, ${highlight.a})`;
        const dimStr = `rgba(${dim.r}, ${dim.g}, ${dim.b}, ${dim.a})`;

        let styleEl = document.getElementById('blur-word-styles');
        if (!styleEl) {
            styleEl = document.createElement('style');
            styleEl.id = 'blur-word-styles';
            document.head.appendChild(styleEl);
        }

        const dimOutline = this.adjustBrightness(this.outlineColor, 0.6);
        const outlineRgba = `rgba(${dimOutline.r}, ${dimOutline.g}, ${dimOutline.b}, ${dimOutline.a})`;

        styleEl.textContent = `
            .blur-word {
                display: inline-block;
                background: linear-gradient(to right, ${highlightStr} 50%, ${dimStr} 50%);
                background-size: 200% 100%;
                background-position: 100% 0;
                -webkit-background-clip: text;
                background-clip: text;
                color: transparent;
                filter: drop-shadow(0 0 0px rgba(${highlight.r},${highlight.g},${highlight.b},0));
                -webkit-text-stroke: ${this.outlineWidth}px ${outlineRgba};
                text-stroke: ${this.outlineWidth}px ${outlineRgba};
            }
        `;
    }

    splitTextToWords(text) {
        if (!text) return [];
        return text.split(/([\u4e00-\u9fff])/g).filter(Boolean).map((w, i) => ({ text: w, time: 0 }));
    }

    createWordTimeline(words, rowElement, lineStartTime) {
        if (!this.params.wordAnimation) return;
        
        const wordEls = rowElement.querySelectorAll('.blur-word');
        if (wordEls.length === 0) return;

        const tl = window.gsap.timeline({ paused: true });
        const textR = this.textColor.r;
        const textG = this.textColor.g;
        const textB = this.textColor.b;
        const maxGlow = this.glowRange;
        let cumStart = 0;

        wordEls.forEach((el, i) => {
            el.style.backgroundPositionX = '100%';
            el.style.filter = `drop-shadow(0 0 0px rgba(${textR},${textG},${textB},0))`;

            const duration = i === 0
                ? (words[0].time - lineStartTime) / 1000
                : (words[i].time - words[i - 1].time) / 1000;
            const adjustedDuration = Math.max(duration, 0.1);
            const startTime = cumStart;

            tl.to(el, { backgroundPositionX: '0%', duration: adjustedDuration, ease: 'none' }, startTime);
            tl.to(el, { filter: `drop-shadow(0 0 ${maxGlow}px rgba(${textR},${textG},${textB},1))`, duration: adjustedDuration, ease: 'none' }, startTime);
            tl.to(el, { scale: 1.05, duration: adjustedDuration, ease: 'none' }, startTime);
            tl.to(el, { filter: `drop-shadow(0 0 ${maxGlow}px rgba(${textR},${textG},${textB},0))`, duration: 1, ease: 'none' }, startTime + adjustedDuration);

            cumStart += adjustedDuration;
        });

        this.wordTimeline = tl;
    }

    getLineHeight() {
        const font = window.configManager?.getModeFont('blur') || {};
        const fontSize = font?.size || '2.4rem';
        const numSize = parseFloat(fontSize) || 2.4;
        const lineSpacing = this.params.lineSpacing ?? 1.5;
        return numSize * lineSpacing * 16;
    }

    render(frameData) {
        const motion = window.motion;
        if (!motion?.lrcData?.length) {
            this.renderPlaceholder();
            return;
        }

        const currentIdx = frameData.currentIndex;
        const { position, isPlaying, isSongEnded } = frameData;
        const visibleLines = this.params.visibleLines || 9;
        const lines = motion.lrcData;

        // 播放结束淡出
        if (isSongEnded && !isPlaying) {
            if (!this.isFadingOut && this.container) {
                this.isFadingOut = true;
                window.gsap.to(this.container, {
                    opacity: 0,
                    duration: 0.4,
                    ease: 'power2.out'
                });
            }
            return;
        }

        // 恢复显示
        if (this.isFadingOut && this.container) {
            this.isFadingOut = false;
            window.gsap.to(this.container, { opacity: 1, duration: 0.2 });
        }

        if (this.lastLineIndex !== currentIdx) {
            const prevIdx = this.lastLineIndex;
            this.lastLineIndex = currentIdx;
            this.renderBlurLines(currentIdx, lines, visibleLines, prevIdx);
        }

        if (this.wordTimeline && this.params.wordAnimation) {
            const line = lines[currentIdx];
            const lineStartTime = line?.time || 0;
            
            if (!isPlaying) {
                const totalDuration = this.wordTimeline.duration() * 1000;
                const progress = Math.max(0, Math.min(1, (position - lineStartTime) / totalDuration));
                this.wordTimeline.progress(progress);
                this.wordTimeline.pause();
            } else {
                if (this.wordTimeline.paused() || !this.wasPlaying) {
                    this.wordTimeline.play();
                }
            }
            this.wasPlaying = isPlaying;
        }

        this.renderInterlude(frameData);
    }

    renderPlaceholder() {
        this.container.innerHTML = '<span class="shadow">等待播放...</span>';
    }

    renderBlurLines(currentIdx, lines, visibleLines, prevIdx) {
        this.lineHeight = this.getLineHeight();
        const halfLines = Math.floor(visibleLines / 2);
        const duration = this.params.scrollDuration || 0.5;

        this.container.style.height = `${this.lineHeight * visibleLines}px`;
        this.container.style.position = 'relative';
        this.container.innerHTML = '';
        this.lineElements = [];

        const startIdx = currentIdx - halfLines;

        for (let i = 0; i < visibleLines; i++) {
            const lineIdx = startIdx + i;
            const lineEl = document.createElement('div');
            lineEl.className = 'blur-line';

            if (lineIdx < 0 || lineIdx >= lines.length) {
                lineEl.textContent = '';
                lineEl.style.opacity = '0';
            } else {
                lineEl.textContent = lines[lineIdx].text;
            }

            const relativeIdx = lineIdx - currentIdx;
            const targetTop = i * this.lineHeight;
            const targetOpacity = this.calculateOpacity(relativeIdx);
            const targetBlur = this.calculateBlur(relativeIdx);
            const targetScale = this.calculateScale(relativeIdx);

            lineEl.style.position = 'absolute';
            lineEl.style.top = targetTop + 'px';
            lineEl.style.textAlign = 'center';
            lineEl.style.willChange = 'transform, opacity, filter';
            lineEl.style.zIndex = (lineIdx === currentIdx) ? '10' : String(visibleLines - Math.abs(i - halfLines));

            // wordAnimation 关闭时，给当前行添加整行发光效果
            const textShadow = (relativeIdx === 0 && this.glowRange > 0 && !this.params.wordAnimation) 
                ? `0 0 ${this.glowRange}px ${this.textColor}` : 'none';
            const textStroke = (relativeIdx === 0 && this.outlineWidth > 0 && !this.params.wordAnimation) 
                ? `${this.outlineWidth}px ${this.outlineColor}` : 'none';

            if (prevIdx !== undefined && prevIdx !== currentIdx) {
                const prevRelativeIdx = lineIdx - prevIdx;
                const prevTop = (prevRelativeIdx + halfLines) * this.lineHeight;
                const prevOpacity = this.calculateOpacity(prevRelativeIdx);
                const prevBlur = this.calculateBlur(prevRelativeIdx);
                const prevScale = this.calculateScale(prevRelativeIdx);

                lineEl.style.top = prevTop + 'px';
                lineEl.style.opacity = prevOpacity;
                lineEl.style.filter = 'blur(' + prevBlur + 'px)';
                lineEl.style.transform = 'translateX(-50%) scale(' + prevScale + ')';
                const prevTextShadow = (prevRelativeIdx === 0 && this.glowRange > 0 && !this.params.wordAnimation) 
                    ? `0 0 ${this.glowRange}px ${this.textColor}` : 'none';
                const prevTextStroke = (prevRelativeIdx === 0 && this.outlineWidth > 0 && !this.params.wordAnimation) 
                    ? `${this.outlineWidth}px ${this.outlineColor}` : 'none';
                lineEl.style.textShadow = prevTextShadow;
                lineEl.style.webkitTextStroke = prevTextStroke;
                lineEl.style.fontWeight = this.fontWeight || 'normal';

                window.gsap.to(lineEl, {
                    top: targetTop,
                    opacity: targetOpacity,
                    scale: targetScale,
                    filter: 'blur(' + targetBlur + 'px)',
                    textShadow: textShadow,
                    webkitTextStroke: textStroke,
                    fontWeight: this.fontWeight || 'normal',
                    duration: duration,
                    ease: 'linear',
                    overwrite: 'auto'
                });
            } else {
                lineEl.style.top = targetTop + 'px';
                lineEl.style.opacity = targetOpacity;
                lineEl.style.filter = 'blur(' + targetBlur + 'px)';
                lineEl.style.transform = 'translateX(-50%) scale(' + targetScale + ')';
                lineEl.style.textShadow = textShadow;
                lineEl.style.webkitTextStroke = textStroke;
                lineEl.style.fontWeight = this.fontWeight || 'normal';
            }

            this.container.appendChild(lineEl);
            this.lineElements.push(lineEl);
            
            if (relativeIdx === 0 && this.params.wordAnimation && lines[lineIdx]?.words) {
                const words = lines[lineIdx].words;
                const textSpan = document.createElement('span');
                textSpan.className = 'blur-line-text';
                textSpan.style.display = 'inline-block';
                words.forEach((word, wi) => {
                    const span = document.createElement('span');
                    span.className = 'blur-word';
                    span.textContent = word.text;
                    textSpan.appendChild(span);
                });
                lineEl.innerHTML = '';
                lineEl.appendChild(textSpan);
                
                this.createWordTimeline(words, lineEl, lines[lineIdx].time);
                if (this.wordTimeline) {
                    this.wordTimeline.play();
                }
            }
        }
    }

    calculateOpacity(relativeIdx) {
        var decay = this.params.opacityDecay;
        if (decay === undefined) decay = 0.15;
        var opacity = 1 - (Math.abs(relativeIdx) * decay);
        return Math.max(0.15, Math.min(1, opacity));
    }

    calculateBlur(relativeIdx) {
        var increment = this.params.blurIncrement;
        if (increment === undefined) increment = 0.5;
        var maxBlur = this.params.blurMax;
        if (maxBlur === undefined) maxBlur = 6;
        var blur = Math.abs(relativeIdx) * increment;
        return Math.min(maxBlur, blur);
    }

    calculateScale(relativeIdx) {
        var decay = this.params.scaleDecay;
        if (decay === undefined) decay = 0;
        var scale = 1 - (Math.abs(relativeIdx) * decay);
        return Math.max(0.3, Math.min(1, scale));
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

    clear() {
        super.clear();
        this.lineElements = [];
        this.wordTimeline = null;
    }

    reset() {
        super.reset();
        this.isFadingOut = false;
        this.lineElements = [];
        this.wordTimeline = null;
        this.currentWords = [];
        if (this.interludeEl) {
            this.interludeEl.remove();
            this.interludeEl = null;
        }
    }

    destroy() {
        super.destroy();
        if (this.interludeEl) {
            this.interludeEl.remove();
            this.interludeEl = null;
        }
        if (this.wordTimeline) {
            this.wordTimeline.kill();
            this.wordTimeline = null;
        }
        const styleEl = document.getElementById('blur-word-styles');
        if (styleEl) styleEl.remove();
    }
}

window.BlurRenderer = BlurRenderer;