class ScrollRenderer extends LyricsRendererBase {
    constructor(container, stage, config) {
        super(container, stage, config);
        this.params = config?.modeParams?.scroll || {};
        
        this.currentHighlightRow = 0;
        this.rowLyrics = [null, null];
        this.rowElements = [null, null];
        this.wordTimelines = [null, null];
        this.currentWords = [null, null];
        
        this.textColor = { r: 255, g: 255, b: 255, a: 1 };
        this.glowRange = 1;
        this.outlineWidth = 1;
        this.outlineColor = { r: 255, g: 255, b: 255, a: 1 };
        this.lastLineIndex = -1;
        this.wasPlaying = true;
    }

    getDefaultConfig() {
        return {
            wordAnimation: true,
            animationDuration: 0.3,
            currentScale: 1.05,
            alignOffset: 0
        };
    }

    applyConfig(config) {
        super.applyConfig(config);
        this.params = config?.modeParams?.scroll || this.getDefaultConfig();
    }

    initStyles() {
        const colors = window.configManager?.getModeColors('scroll') || {};
        const font = window.configManager?.getModeFont('scroll') || {};
        
        this.glowRange = colors.glowRange ?? 1;
        this.outlineWidth = colors.outlineWidth ?? 1;
        this.outlineColor = this.parseColor(colors.outlineColor || '#ffffff');
        this.textColor = this.parseColor(colors.text || '#ffffff');

        document.documentElement.style.setProperty('--fg', colors.text || '#ffffff');
        
        const fontFamily = font?.family || 'system-ui, -apple-system, Arial';
        const fontSize = font?.size || '2.4rem';
        const fontWeight = font?.weight || 'normal';
        document.documentElement.style.setProperty('--font-family', fontFamily);
        document.documentElement.style.setProperty('--font-size', fontSize);
        document.documentElement.style.setProperty('--font-weight', fontWeight);
        this.fontWeight = fontWeight;

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

        let styleEl = document.getElementById('scroll-word-styles');
        if (!styleEl) {
            styleEl = document.createElement('style');
            styleEl.id = 'scroll-word-styles';
            document.head.appendChild(styleEl);
        }

        const dimOutline = this.adjustBrightness(this.outlineColor, 0.6);
        const outlineRgba = `rgba(${dimOutline.r}, ${dimOutline.g}, ${dimOutline.b}, ${dimOutline.a})`;

        styleEl.textContent = `
            .scroll-word {
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
            .scroll-row-dim { opacity: 0.6; }
        `;
    }

    splitTextToWords(text) {
        if (!text) return [];
        return text.split(/([\u4e00-\u9fff])/g).filter(Boolean).map((w, i) => ({ text: w, time: 0 }));
    }

    createWordTimeline(words, rowIndex, lineStartTime) {
        if (!this.params.wordAnimation) {
            this.wordTimelines[rowIndex] = null;
            return;
        }

        const row = this.rowElements[rowIndex];
        if (!row) return;

        const wordEls = row.querySelectorAll('.scroll-word');
        if (wordEls.length === 0) {
            this.wordTimelines[rowIndex] = null;
            return;
        }

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
            tl.to(el, { scale: 1, duration: 1, ease: 'none' }, startTime + adjustedDuration);

            cumStart += adjustedDuration;
        });

        this.wordTimelines[rowIndex] = tl;
    }

    renderRow(text, rowIndex, isDim) {
        const words = this.splitTextToWords(text);
        this.currentWords[rowIndex] = words;
        const row = document.createElement('div');
        row.className = 'scroll-row' + (isDim ? ' scroll-row-dim' : '');
        row.style.fontWeight = this.fontWeight || 'normal';
        
        // 对齐偏移：上行(rowIndex=0)与alignOffset同向，下行(rowIndex=1)与alignOffset反向
        const offset = this.params.alignOffset || 0;
        const rowOffset = rowIndex === 0 ? offset : -offset;
        if (rowOffset !== 0) {
            row.style.marginLeft = rowOffset + 'px';
        }
        
        // wordAnimation 关闭时，给高亮行添加发光效果
        if (!isDim && this.glowRange > 0 && !this.params.wordAnimation) {
            const textR = this.textColor.r;
            const textG = this.textColor.g;
            const textB = this.textColor.b;
            row.style.textShadow = `0 0 ${this.glowRange}px rgba(${textR},${textG},${textB},1)`;
        }
        
        words.forEach((word, i) => {
            const span = document.createElement('span');
            span.className = 'scroll-word';
            span.textContent = word.text;
            row.appendChild(span);
        });
        this.rowElements[rowIndex] = row;
        return row;
    }

    render(frameData) {
        const motion = window.motion;
        if (!motion?.lrcData?.length) {
            this.renderPlaceholder();
            return;
        }

        const currentIdx = frameData.currentIndex;
        const { position, isPlaying, isSongEnded } = frameData;
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

        if (this.lastLineIndex === -1) {
            this.lastLineIndex = currentIdx;
            this.rowLyrics[0] = lines[currentIdx]?.text || '';
            this.rowLyrics[1] = lines[currentIdx + 1]?.text || '';
            
            this.container.innerHTML = '';
            const row0 = this.renderRow(this.rowLyrics[0], 0, false);
            const row1 = this.renderRow(this.rowLyrics[1], 1, true);
            this.container.appendChild(row0);
            this.container.appendChild(row1);
            
            this.currentHighlightRow = 0;
            
            const line0 = lines[currentIdx];
            if (line0?.words && line0.words.length > 0) {
                this.createWordTimeline(line0.words, 0, line0.time);
                this.currentWords[0] = line0.words;
            } else {
                this.createWordTimeline(this.currentWords[0], 0, 0);
            }
            
            if (this.wordTimelines[0]) {
                this.wordTimelines[0].play();
            }
            
            return;
        }

        if (currentIdx !== this.lastLineIndex) {
            this.lastLineIndex = currentIdx;
            const highlightIdx = this.currentHighlightRow;
            const dimIdx = 1 - highlightIdx;
            
            const nextLine = lines[currentIdx + 1];
            const nextNextLine = lines[currentIdx + 2];
            
            if (!nextLine) {
                this.container.innerHTML = '';
                const row0 = this.renderRow(this.rowLyrics[0], 0, this.currentHighlightRow !== 0);
                const row1 = this.renderRow(this.rowLyrics[1], 1, this.currentHighlightRow !== 1);
                this.container.appendChild(row0);
                this.container.appendChild(row1);
                return;
            }
            
            this.rowLyrics[highlightIdx] = nextNextLine?.text || this.rowLyrics[dimIdx];
            this.rowLyrics[dimIdx] = nextLine.text || '';
            this.currentHighlightRow = dimIdx;
            
            this.container.innerHTML = '';
            const row0 = this.renderRow(this.rowLyrics[0], 0, this.currentHighlightRow !== 0);
            const row1 = this.renderRow(this.rowLyrics[1], 1, this.currentHighlightRow !== 1);
            this.container.appendChild(row0);
            this.container.appendChild(row1);
            
            const line = lines[currentIdx + 1];
            if (line?.words && line.words.length > 0) {
                this.createWordTimeline(line.words, this.currentHighlightRow, line.time);
                this.currentWords[this.currentHighlightRow] = line.words;
            } else {
                this.createWordTimeline(this.currentWords[this.currentHighlightRow], this.currentHighlightRow, 0);
            }
            
            if (this.wordTimelines[this.currentHighlightRow]) {
                this.wordTimelines[this.currentHighlightRow].play();
            }
            
            return;
        }

        const highlightRow = this.currentHighlightRow;
        if (this.wordTimelines[highlightRow]) {
            const line = lines[currentIdx + 1];
            const lineStartTime = line?.time || 0;
            
            if (!isPlaying) {
                const totalDuration = this.wordTimelines[highlightRow].duration() * 1000;
                const progress = Math.max(0, Math.min(1, (position - lineStartTime) / totalDuration));
                this.wordTimelines[highlightRow].progress(progress);
                this.wordTimelines[highlightRow].pause();
            } else {
                if (this.wordTimelines[highlightRow].paused() || !this.wasPlaying) {
                    this.wordTimelines[highlightRow].play();
                }
            }
            this.wasPlaying = isPlaying;
        }

        this.renderInterlude(frameData);
    }

    renderPlaceholder() {
        this.container.innerHTML = '<span class="shadow">等待播放...</span>';
    }

    renderInterlude(frameData) {
        if (frameData.isInterlude && frameData.countdown > 0) {
            if (!this.interludeEl) {
                this.interludeEl = document.createElement('div');
                this.interludeEl.id = 'interlude';
                this.interludeEl.style.cssText = 'position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); font-size: 4rem; font-weight: bold; color: rgba(255,255,255,0.3); text-shadow: 0 0 20px rgba(0,255,255,0.5);';
                this.stage.appendChild(this.interludeEl);
            }
            this.interludeEl.textContent = frameData.countdown;
            window.gsap.fromTo(this.interludeEl, { scale: 1.3, opacity: 0.5 }, { scale: 1, opacity: 1, duration: 0.5, ease: 'back.out(1.7)' });
        } else if (this.interludeEl) {
            window.gsap.to(this.interludeEl, { opacity: 0, duration: 0.3, onComplete: () => { this.interludeEl?.remove(); this.interludeEl = null; } });
        }
    }

    clear() {
        super.clear();
        this.rowElements = [null, null];
        this.wordTimelines = [null, null];
    }

    reset() {
        super.reset();
        this.isFadingOut = false;
        this.rowLyrics = ['', ''];
        this.rowElements = [null, null];
        this.currentHighlightRow = 0;
        this.wordTimelines = [null, null];
        this.currentWords = [[], []];
        if (this.interludeEl) {
            this.interludeEl.remove();
            this.interludeEl = null;
        }
    }

    destroy() {
        super.destroy();
        if (this.interludeEl) { this.interludeEl.remove(); this.interludeEl = null; }
        const styleEl = document.getElementById('scroll-word-styles');
        if (styleEl) styleEl.remove();
    }
}

window.ScrollRenderer = ScrollRenderer;