class RendererManager {
    constructor() {
        this.currentRenderer = null;
        this.container = null;
        this.stage = null;
        this.config = null;
        this.motion = null;
    }

    async init(containerId, stageId) {
        this.container = document.getElementById(containerId || 'lyrics');
        this.stage = document.getElementById(stageId || 'lyricsStage');

        await window.configManager.load();
        this.config = window.configManager.get();

        this.initStyles();
        this.createRenderer();
        this.bindMotion();
    }

    initStyles() {
        const colors = this.config?.colors || {};
        const font = this.config?.font || {};
        const bg = this.config?.bg || {};

        const isOBS = navigator.userAgent.includes('OBSBrowser') || window.obsstudio;
        const bgColor = bg?.color || '#000000';

        if (!isOBS) {
            document.documentElement.style.setProperty('--bg', bgColor);
            document.body.style.background = bgColor;
            if (this.stage) {
                this.stage.style.background = bgColor;
            }
        }

        document.documentElement.style.setProperty('--fg', colors?.text || '#ffffff');

        // 设置字体
        const fontFamily = font?.family || 'system-ui, -apple-system, Arial';
        const fontSize = font?.size || '2.4rem';
        document.documentElement.style.setProperty('--font-family', fontFamily);
        document.documentElement.style.setProperty('--font-size', fontSize);

        // 应用到 body 和 stage
        document.body.style.fontFamily = fontFamily;
        document.body.style.fontSize = fontSize;
        if (this.stage) {
            this.stage.style.fontFamily = fontFamily;
            this.stage.style.fontSize = fontSize;
        }
        if (this.container) {
            this.container.style.fontFamily = fontFamily;
            this.container.style.fontSize = fontSize;
        }
    }

    createRenderer() {
        const mode = this.config?.mode || 'karaoke';

        if (this.currentRenderer) {
            this.currentRenderer.destroy();
        }

        switch (mode) {
            case 'scroll':
                this.currentRenderer = new window.ScrollRenderer(this.container, this.stage, this.config);
                break;
            case 'blur':
                this.currentRenderer = new window.BlurRenderer(this.container, this.stage, this.config);
                break;
            case 'karaoke':
            default:
                this.currentRenderer = new window.KaraokeRenderer(this.container, this.stage, this.config);
                break;
        }

        if (this.currentRenderer) {
            this.currentRenderer.initStyles();
        }
    }

    bindMotion() {
        this.motion = window.motion;
        if (this.motion) {
            this.motion.onFrame = (frameData) => {
                if (this.currentRenderer) {
                    this.currentRenderer.render(frameData);
                }
            };
            this.motion.onConnectionChange = (connected) => {
                if (this.currentRenderer?.showConnecting) {
                    this.currentRenderer.showConnecting(!connected);
                }
            };
        }
    }

    async reloadConfig() {
        await window.configManager.load();
        this.config = window.configManager.get();
        this.initStyles();
        this.createRenderer();
    }

    async switchMode(mode) {
        this.config = { ...this.config, mode };
        await window.configManager.save(this.config);
        this.initStyles();
        this.createRenderer();
    }

    async updateConfig(newConfig) {
        this.config = { ...this.config, ...newConfig };
        await window.configManager.save(this.config);
        this.initStyles();
        this.createRenderer();
    }

    getRenderer() {
        return this.currentRenderer;
    }

    getConfig() {
        return this.config;
    }

    destroy() {
        if (this.currentRenderer) {
            this.currentRenderer.destroy();
            this.currentRenderer = null;
        }
    }
}

window.RendererManager = RendererManager;
window.rendererManager = new RendererManager();