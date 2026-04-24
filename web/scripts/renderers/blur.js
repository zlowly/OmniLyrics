class BlurRenderer extends LyricsRendererBase {
    constructor(container, stage, config) {
        super(container, stage, config);
        this.params = config?.modeParams?.blur || {};
        this.lineElements = [];
    }

    getDefaultConfig() {
        return {
            visibleLines: 9,
            lineSpacing: 1.5,
            opacityDecay: 0.15,
            blurIncrement: 0.5,
            scaleDecay: 0,
            blurMax: 6,
            scrollSpeed: 'linear',
            scrollDuration: 0.5
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
        document.documentElement.style.setProperty('--font-family', fontFamily);
        document.documentElement.style.setProperty('--font-size', fontSize);
        
        this.textColor = colors.text || '#ffffff';
        this.glowRange = colors.glowRange ?? 1;
        this.outlineWidth = colors.outlineWidth ?? 1;
        this.outlineColor = colors.outlineColor || '#ffffff';
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
        const visibleLines = this.params.visibleLines || 9;
        const lines = motion.lrcData;

        if (this.lastLineIndex !== currentIdx) {
            const prevIdx = this.lastLineIndex;
            this.lastLineIndex = currentIdx;
            this.renderBlurLines(currentIdx, lines, visibleLines, prevIdx);
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

            const textShadow = (relativeIdx === 0 && this.glowRange > 0) ? `0 0 ${this.glowRange}px ${this.textColor}` : 'none';
            const textStroke = (relativeIdx === 0 && this.outlineWidth > 0) ? `${this.outlineWidth}px ${this.outlineColor}` : 'none';

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
                lineEl.style.textShadow = (prevRelativeIdx === 0 && this.glowRange > 0) ? `0 0 ${this.glowRange}px ${this.textColor}` : 'none';
                lineEl.style.webkitTextStroke = (prevRelativeIdx === 0 && this.outlineWidth > 0) ? `${this.outlineWidth}px ${this.outlineColor}` : 'none';

                window.gsap.to(lineEl, {
                    top: targetTop,
                    opacity: targetOpacity,
                    scale: targetScale,
                    filter: 'blur(' + targetBlur + 'px)',
                    textShadow: textShadow,
                    webkitTextStroke: textStroke,
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
            }

            this.container.appendChild(lineEl);
            this.lineElements.push(lineEl);
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
    }

    destroy() {
        super.destroy();
        if (this.interludeEl) {
            this.interludeEl.remove();
            this.interludeEl = null;
        }
    }
}

window.BlurRenderer = BlurRenderer;