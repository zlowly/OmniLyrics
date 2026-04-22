// 从当前页面 URL 自动提取主机和端口，用于 API 调用
const CONFIG_API_BASE = (() => {
    const loc = window.location;
    return `${loc.protocol}//${loc.host}`;
})();

const SETTINGS_DEFAULT_CONFIG = {
    mode: 'karaoke',
    colors: {
        text: '#ffffff',
        bg: '#000000',
        glowRange: 1,
        outlineWidth: 1,
        outlineColor: '#ffffff'
    },
    font: {
        size: '2.4rem',
        family: 'system-ui, -apple-system, Arial'
    },
    bg: {
        color: '#000000'
    },
    modeParams: {
        karaoke: {
            wordAnimation: true,
            animationDuration: 0.3,
            currentScale: 1.05
        },
        scroll: {
            showNext: true,
            nextOpacity: 0.6,
            scrollDuration: 0.4
        },
        blur: {
            visibleLines: 9,
            lineSpacing: 1.5,
            opacityDecay: 0.15,
            blurIncrement: 0.5,
            scaleDecay: 0.1,
            blurMax: 6,
            scrollSpeed: 'linear',
            scrollDuration: 0.5
        }
    }
};

class ConfigManager {
    constructor() {
        this.config = null;
        this.loaded = false;
    }

    async load() {
        try {
            const res = await fetch(`${CONFIG_API_BASE}/config`);
            if (res.ok) {
                this.config = await res.json();
                this.loaded = true;
            } else {
                this.config = SETTINGS_DEFAULT_CONFIG;
                this.loaded = true;
            }
        } catch (e) {
            console.warn('[Config] Failed to load, using default:', e);
            this.config = SETTINGS_DEFAULT_CONFIG;
            this.loaded = true;
        }
        return this.config;
    }

    async save(config) {
        try {
            const res = await fetch(`${CONFIG_API_BASE}/config`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(config)
            });
            if (res.ok) {
                this.config = config;
                return true;
            }
        } catch (e) {
            console.warn('[Config] Failed to save:', e);
        }
        return false;
    }

    async reset() {
        this.config = SETTINGS_DEFAULT_CONFIG;
        return this.save(this.config);
    }

    get() {
        return this.config || SETTINGS_DEFAULT_CONFIG;
    }

    getMode() {
        return this.config?.mode || SETTINGS_DEFAULT_CONFIG.mode;
    }

    getColors() {
        return this.config?.colors || SETTINGS_DEFAULT_CONFIG.colors;
    }

    getFont() {
        return this.config?.font || SETTINGS_DEFAULT_CONFIG.font;
    }

    getBg() {
        return this.config?.bg || SETTINGS_DEFAULT_CONFIG.bg;
    }

    getModeParams(mode) {
        return this.config?.modeParams?.[mode] || SETTINGS_DEFAULT_CONFIG.modeParams[mode];
    }
}

const configManager = new ConfigManager();
window.configManager = configManager;