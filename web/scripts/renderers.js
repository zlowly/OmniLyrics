// renderers.js - 渲染层：视觉插件系统
// 职责：OBS 优化样式、逐字发光、3D 拨盘、动态模糊

const motion = window.motion;
const gsap = window.gsap;

class LyricsRenderer {
    constructor() {
        this.container = document.getElementById('lyrics');
        this.stage = document.getElementById('lyricsStage');
        this.currentWords = [];
        this.wordElements = [];
        this.initStyles();
    }

    initStyles() {
        document.documentElement.style.setProperty('--bg', 'rgba(0,0,0,0)');
        document.documentElement.style.setProperty('--fg', '#ffffff');
        document.documentElement.style.setProperty('--glow', 'rgba(0,255,255,.9)');
    }

    render(frameData) {
        const hasNoLyrics = !motion.lrcData.length && motion.fetchFailed;
        if (hasNoLyrics) {
            this.renderNoLyrics();
            return;
        }

        if (!motion.lrcData.length) {
            this.renderPlaceholder();
            return;
        }

        const line = motion.lrcData[frameData.currentIndex] || motion.lrcData[0];
        if (!line) return;

        if (this.lastLineIndex !== frameData.currentIndex) {
            this.lastLineIndex = frameData.currentIndex;
            this.renderNewLine(line.text);
        }

        this.updateWordProgress(frameData.lineProgress);
        this.applyMotionBlur(frameData.velocity);
        this.renderInterlude(frameData);
    }

    renderPlaceholder() {
        this.container.innerHTML = '<span class="shadow">等待播放...</span>';
    }

    renderNoLyrics() {
        this.container.innerHTML = '<span class="shadow" style="color: #ff6b6b;">无法获取歌词</span>';
    }

    renderNewLine(text) {
        const words = text.split(/(?=[^\x00-\xff])/).filter(w => w.trim());
        this.currentWords = words;
        this.container.innerHTML = '';
        this.wordElements = [];

        words.forEach((word, i) => {
            const span = document.createElement('span');
            span.className = 'word';
            span.textContent = word;
            span.dataset.index = i;
            span.dataset.visible = 'false';
            this.container.appendChild(span);
            this.wordElements.push(span);
        });

        gsap.fromTo(this.container,
            { opacity: 0, y: 20, rotateX: -15 },
            { opacity: 1, y: 0, rotateX: 0, duration: 0.4, ease: 'power2.out' }
        );
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
                if (isVisible) {
                    this.animateWordGlow(el, isCurrent);
                }
            }
        });
    }

    animateWordGlow(el, isCurrent) {
        gsap.to(el, {
            '--g': isCurrent ? 1.5 : 1,
            textShadow: `0 0 ${isCurrent ? 12 : 6}px rgba(0,255,255,${isCurrent ? 0.9 : 0.6})`,
            duration: 0.3,
            ease: 'power1.out'
        });

        if (isCurrent) {
            gsap.fromTo(el,
                { scale: 1, z: 0 },
                { scale: 1.05, z: 5, duration: 0.15, yoyo: true, repeat: 1 }
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
            gsap.fromTo(this.interludeEl, 
                { scale: 1.3, opacity: 0.5 },
                { scale: 1, opacity: 1, duration: 0.5, ease: 'back.out(1.7)' }
            );
        } else if (this.interludeEl) {
            gsap.to(this.interludeEl, {
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
            gsap.to(this.connEl, {
                opacity: 0, duration: 0.5, onComplete: () => {
                    this.connEl.style.display = 'none';
                }
            });
        }
    }
}

const renderer = new LyricsRenderer();

if (window.motion) {
    window.motion.onFrame = (frameData) => renderer.render(frameData);
    window.motion.onConnectionChange = (connected) => renderer.showConnecting(!connected);
}

window.addEventListener('online', () => { if(window.motion) window.motion.online = true; });
window.addEventListener('offline', () => { if(window.motion) window.motion.online = false; });