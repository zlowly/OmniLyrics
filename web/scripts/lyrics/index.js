// 歌词源调度器
// 功能：按优先级顺序/并行搜索，多源调度，错误处理

const LYRICS_API_BASE = (() => {
    const loc = window.location;
    return `${loc.protocol}//${loc.host}`;
})();

class LyricsScheduler {
    constructor() {
        this.config = null;
        this.providers = new Map();
    }

    init(providerInstances) {
        for (const p of providerInstances) {
            this.providers.set(p.name, p);
        }
        // Auto-register kgmusic provider if available and not already registered
        try {
            if (!this.providers.has('kgmusic') && window.KGMusicProvider) {
                const kgProvider = new window.KGMusicProvider();
                kgProvider.name = 'kgmusic';
                this.providers.set(kgProvider.name, kgProvider);
                console.log('[Lyrics] KGMusicProvider auto-registered');
            }
        } catch (e) {
            console.warn('[Lyrics] KGMusicProvider auto-register failed:', e);
        }
        console.log('[Lyrics] Scheduler initialized with providers:', [...this.providers.keys()]);
    }

    async loadConfig() {
        try {
            const resp = await fetch(`${LYRICS_API_BASE}/config/lyrics`);
            if (resp.ok) {
                this.config = await resp.json();
                console.log('[Lyrics] Config loaded:', this.config);
                return true;
            }
        } catch (e) {
            console.warn('[Lyrics] Config load failed:', e);
        }
        return false;
    }

    async search(title, artist, duration, appName) {
        if (!this.config) {
            await this.loadConfig();
        }

        if (!this.config) {
            console.warn('[Lyrics] No config, using fallback');
            return this.searchFallback(title, artist, duration);
        }

        const timeout = this.config.timeout || 5000;
        const groups = this.groupByPriority(this.config.sources || [], appName);

        for (const group of groups) {
            const result = await this.searchGroupParallel(group, title, artist, duration, timeout);
            if (result) {
                return result;
            }
        }

        return null;
    }

    searchFallback(title, artist, duration) {
        return this.searchSingleSource('lrclib', title, artist, duration);
    }

    searchSingleSource(name, title, artist, duration) {
        const provider = this.providers.get(name);
        if (!provider) return null;
        return provider.search(title, artist, duration);
    }

    groupByPriority(sources, appName) {
        if (!sources || sources.length === 0) return [];

        // 找出显式指定该 appName 的源（不包括只有 * 的源）
        const explicitMatches = sources.filter(s =>
            s.enabled &&
            s.apps && s.apps.length > 0 &&
            s.apps.includes(appName) &&
            !s.apps.every(a => a === '*')
        );

        // 如果有显式匹配的源，按 priority 排序
        if (explicitMatches.length > 0) {
            return this.buildGroups(explicitMatches.sort((a, b) => a.priority - b.priority));
        }

        // 没有显式匹配时，使用 * 通用源
        const wildcardSources = sources.filter(s =>
            s.enabled && s.apps && s.apps.includes('*')
        );
        return this.buildGroups(wildcardSources.sort((a, b) => a.priority - b.priority));
    }

    buildGroups(sortedSources) {
        const groups = [];
        for (const source of sortedSources) {
            const lastGroup = groups[groups.length - 1];
            if (lastGroup && lastGroup[0].priority === source.priority) {
                lastGroup.push(source);
            } else {
                groups.push([source]);
            }
        }
        return groups;
    }

    matchApps(apps, appName) {
        if (!apps || apps.length === 0) return false;
        return apps.includes('*') || apps.includes(appName);
    }

    async searchGroupParallel(group, title, artist, duration, timeout) {
        const promises = group.map(source => {
            return this.searchWithTimeout(source.name, title, artist, duration, timeout);
        });

        const results = await Promise.allSettled(promises);

        let bestResult = null;
        let hasWordTimestamp = false;

        for (const r of results) {
            if (r.status !== 'fulfilled') continue;
            const data = r.value;
            if (!data) continue;

            const hasWords = this.hasWordTimestamp(data);
            if (!hasWordTimestamp) {
                if (hasWords) {
                    bestResult = data;
                    hasWordTimestamp = true;
                } else if (!bestResult) {
                    bestResult = data;
                }
            }
        }

        return bestResult;
    }

    async searchWithTimeout(name, title, artist, duration, timeout) {
        const provider = this.providers.get(name);
        if (!provider) return null;

        try {
            const result = await Promise.race([
                provider.search(title, artist, duration),
                new Promise((_, reject) => setTimeout(() => reject(new Error('timeout')), timeout))
            ]);
            return result;
        } catch (e) {
            console.warn(`[Lyrics] ${name} failed:`, e.message);
            return null;
        }
    }

    hasWordTimestamp(data) {
        if (!data?.lrcData) return false;
        return data.lrcData.some(line => line.words && line.words.length > 1);
    }
}

// 注册到全局
window.LyricsScheduler = LyricsScheduler;

// 自动初始化所有 providers（包含 kgmusic）
try {
    const providers = [];
    if (window.LRCLibProvider) providers.push(new window.LRCLibProvider());
    if (window.QQMusicProvider) providers.push(new window.QQMusicProvider());
    if (window.KGMusicProvider) providers.push(new window.KGMusicProvider());

    const scheduler = new LyricsScheduler();
    scheduler.init(providers);
    window.lyricsScheduler = scheduler;  // also export as lowercase for motion.js
} catch (e) {
    console.warn('[Lyrics] Provider init failed:', e);
}
