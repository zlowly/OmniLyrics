class LyricsRendererBase {
    constructor(container, stage, config) {
        this.container = container;
        this.stage = stage;
        this.config = config || {};
        this.lastLineIndex = -1;
        this.currentWords = [];
        this.wordElements = [];
    }

    render(frameData) {
        throw new Error('render() must be implemented by subclass');
    }

    getDefaultConfig() {
        return {};
    }

    applyConfig(config) {
        this.config = config;
    }

    initStyles() {
        throw new Error('initStyles() must be implemented by subclass');
    }

    detectInterlude(text) {
        const keywords = ['间奏', '器乐', 'instrumental', 'interlude', '空白', '...'];
        return keywords.some(k => text.toLowerCase().includes(k.toLowerCase()));
    }

    clear() {
        if (this.container) {
            this.container.innerHTML = '';
        }
        this.wordElements = [];
        this.currentWords = [];
        this.lastLineIndex = -1;
    }

    destroy() {
        this.clear();
    }
}

window.LyricsRendererBase = LyricsRendererBase;