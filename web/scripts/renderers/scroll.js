class ScrollRenderer extends LyricsRendererBase {
    constructor(container, stage, config) {
        super(container, stage, config);
        this.params = config?.modeParams?.scroll || {};
        this.currentLineEl = null;
        this.nextLineEl = null;
    }

    getDefaultConfig() {
        return {
            showNext: true,
            nextOpacity: 0.6,
            scrollDuration: 0.4
        };
    }

    applyConfig(config) {
        super.applyConfig(config);
        this.params = config?.modeParams?.scroll || this.getDefaultConfig();
    }

    initStyles() {
        const colors = window.configManager?.getColors() || {};
        document.documentElement.style.setProperty('--fg', colors.text || '#ffffff');
    }

    render(frameData) {
        const motion = window.motion;
        if (!motion?.lrcData?.length) {
            this.renderPlaceholder();
            return;
        }

        const currentIdx = frameData.currentIndex;
        const nextIdx = currentIdx + 1;
        const lines = motion.lrcData;

        if (this.lastLineIndex !== currentIdx) {
            this.lastLineIndex = currentIdx;
            this.renderScroll(lines[currentIdx]?.text || '', lines[nextIdx]?.text || '');
        }

        this.renderInterlude(frameData);
    }

    renderPlaceholder() {
        this.container.innerHTML = '<span class="shadow">等待播放...</span>';
    }

    renderScroll(currentText, nextText) {
        this.container.innerHTML = '';
        this.wordElements = [];

        this.currentLineEl = document.createElement('div');
        this.currentLineEl.className = 'scroll-current';
        this.currentLineEl.textContent = currentText;
        this.container.appendChild(this.currentLineEl);

        if (this.params.showNext && nextText) {
            this.nextLineEl = document.createElement('div');
            this.nextLineEl.className = 'scroll-next';
            this.nextLineEl.textContent = nextText;
            this.nextLineEl.style.opacity = this.params.nextOpacity || 0.6;
            this.container.appendChild(this.nextLineEl);
        }

        const duration = this.params.scrollDuration || 0.4;
        window.gsap.fromTo(this.container,
            { opacity: 0, y: 10 },
            { opacity: 1, y: 0, duration, ease: 'power2.out' }
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

    clear() {
        super.clear();
        this.currentLineEl = null;
        this.nextLineEl = null;
    }

    destroy() {
        super.destroy();
        if (this.interludeEl) {
            this.interludeEl.remove();
            this.interludeEl = null;
        }
    }
}

window.ScrollRenderer = ScrollRenderer;