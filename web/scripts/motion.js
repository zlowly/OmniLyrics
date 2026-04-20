// motion.js - 驱动层：逻辑运动引擎
// 职责：心跳监测、多源竞态调度、状态计算

const API_BASE = 'http://localhost:8080';
const LRCLIB_API = 'https://lrclib.net/api/get';

async function convertToTraditional(text) {
    if (!text) return text;
    try {
        const res = await fetch(
            `https://api.zhconvert.org/convert?text=${encodeURIComponent(text)}&converter=Traditional`
        );
        const json = await res.json();
        return json?.data?.text || text;
    } catch (e) {
        console.warn('[Convert] Failed:', e);
        return text;
    }
}

async function searchLrclib(title, artist, currentDuration) {
    const searchUrl = `${LRCLIB_API.replace('/get', '/search')}?track_name=${encodeURIComponent(title)}&artist_name=${encodeURIComponent(artist || '')}`;
    const res = await fetch(searchUrl);
    if (!res.ok) return null;
    const results = await res.json();
    if (!Array.isArray(results) || results.length === 0) return null;

    const withDuration = results.filter(r => r.duration && r.duration > 0);
    const best = withDuration.length === 0
        ? results[0]
        : withDuration.sort((a, b) => Math.abs(a.duration - currentDuration) - Math.abs(b.duration - currentDuration))[0];

    const getUrl = `${LRCLIB_API.replace('/get', '/get')}/${best.id}`;
    const getRes = await fetch(getUrl);
    if (!getRes.ok) return null;
    const json = await getRes.json();
    if (!json?.syncedLyrics && !json?.plainLyrics) return null;

    const lyrics = json.syncedLyrics || json.plainLyrics;
    return {
        lrcData: parseLRCInternal(lyrics),
        lyrics
    };
}

function parseLRCInternal(lrcText) {
    if (!lrcText || typeof lrcText !== 'string') return [];
    lrcText = lrcText.replace(/^\uFEFF/, '');
    const lines = [];
    const timeRegex = /\[(\d{2}):(\d{2})\.(\d{2,3})\]/;
    const cleaned = cleanLRCInternal(lrcText);

    const rawLines = cleaned.split('\n');
    for (const rawLine of rawLines) {
        const trimmed = rawLine.trim();
        if (!trimmed) continue;
        const match = timeRegex.exec(trimmed);
        if (!match) continue;
        const min = parseInt(match[1]);
        const sec = parseInt(match[2]);
        const ms = match[3].length === 3 ? parseInt(match[3]) : parseInt(match[3]) * 10;
        const timeMs = (min * 60 + sec) * 1000 + ms;
        const text = trimmed.substring(match[0].length).trim();
        if (text) lines.push({ time: timeMs, text });
    }
    return lines.sort((a, b) => a.time - b.time);
}

function cleanLRCInternal(lrc) {
    if (!lrc || typeof lrc !== 'string') return '';
    const noisePatterns = [
        /^\s*制作[:：]/i, /^\s*发行[:：]/i, /^\s*专辑[:：]/i,
        /^\s*歌词[:：]/i, /^\s*编曲[:：]/i, /^\s*混音[:：]/i
    ];
    let cleaned = lrc;
    noisePatterns.forEach(p => { cleaned = cleaned.replace(p, ''); });
    return cleaned;
}

const lyricsSources = [
    { name: 'lrclib', search: searchLrclib }
];

class MotionEngine {
    constructor() {
        this.isConnected = false;
        this.lrcData = [];
        this.currentIndex = 0;
        this.lastPosition = 0;
        this.lastTitle = '';
        this.velocity = 0;
        this.frameData = {
            currentIndex: 0,
            lineProgress: 0,
            isInterlude: false,
            countdown: 0,
            velocity: 0
        };
        this.onFrame = null;
        this.onConnectionChange = null;
        this.online = navigator.onLine;
        this.fetchFailed = false;
        this.init();
    }

    async init() {
        this.startHeartbeat();
        this.startRenderLoop();
    }

    startHeartbeat() {
        setInterval(async () => {
            try {
                const res = await fetch(`${API_BASE}/status`, { cache: 'no-store' });
                if (res.ok) {
                    const data = await res.json();
                    if (!this.isConnected) {
                        this.isConnected = true;
                        this.onConnectionChange?.(true);
                    }
                    await this.processBackendData(data);
                } else {
                    this.handleDisconnect();
                }
            } catch (e) {
                this.handleDisconnect();
            }
        }, 200);
    }

    handleDisconnect() {
        if (this.isConnected) {
            this.isConnected = false;
            this.onConnectionChange?.(false);
        }
    }

    async processBackendData(data) {
        const position = data.position || 0;
        const duration = data.duration || 0;

        // 检测换歌，重置重试计数和歌词
        if (data.title !== this.lastTitle) {
            this.lastTitle = data.title;
            this.lrcData = [];
            fetchRetryCount = 0;
            this.fetchFailed = false;
            console.log('[Motion] New song:', data.title, 'duration:', duration);

            // 新歌曲时检查本地缓存
            await this.checkLocalCache(data.title, data.artist, duration);
        }

        this.lastPosition = position;
    }

async checkLocalCache(title, artist, duration) {
        if (!title) return;
        try {
            const url = `${API_BASE}/check_cache?title=${encodeURIComponent(title)}&artist=${encodeURIComponent(artist || '')}`;
            const res = await fetch(url);
            if (res.ok) {
                const result = await res.json();
                let lrcContent = typeof result.content === 'string' ? result.content : result.content?.value || '';
                lrcContent = lrcContent.replace(/\\n/g, '\n').replace(/\\r/g, '');
                if (result.found && lrcContent) {
                    console.log('[Motion] Found local cache, parsing...');
                    this.lrcData = parseLRCInternal(lrcContent);
                    console.log('[Motion] Parsed lrcData length:', this.lrcData.length, this.lrcData.slice(0, 3));
                    // 写入后端缓存（让后端记住有缓存）
                    this.lastLrcContent = lrcContent;
                } else if (this.online && !this.fetchFailed) {
                    // 无本地缓存，获取在线歌词（duration单位从毫秒转为秒）
                    await this.fetchOnlineLyrics(title, artist, Math.floor(duration / 1000));
                }
            }
        } catch (e) {
            console.warn('[Motion] checkLocalCache failed:', e);
        }
    }

parseLRC(lrcText) {
        return parseLRCInternal(lrcText);
    }

    cleanLRC(lrc) {
        return cleanLRCInternal(lrc);
    }

    shouldFetchOnline(data) {
        const local = this.lrcData.find(l => l.text.includes('[偏移:'));
        const hasWordLevel = this.lrcData.some(l => /\[.*?\d+:\d+\.\d+\]/.test(JSON.stringify(l)));
        return !local && hasWordLevel;
    }

    async fetchOnlineLyrics(title, artist, currentDuration) {
        if (!title || !this.online) return;
        if (fetchRetryCount >= MAX_RETRY) {
            console.warn('[Lyrics] Max retries reached, giving up');
            this.fetchFailed = true;
            return;
        }

        let lastError = null;

        for (const source of lyricsSources) {
            try {
                let result = await source.search(title, artist, currentDuration);
                if (result) {
                    console.log(`[Lyrics] Found from ${source.name}:`, title, artist);
                    this.lrcData = result.lrcData;
                    this.updateCache(title, artist, result.lyrics);
                    fetchRetryCount = 0;
                    return;
                }

                const titleTw = await convertToTraditional(title);
                const artistTw = await convertToTraditional(artist);
                if (titleTw !== title || artistTw !== artist) {
                    result = await source.search(titleTw, artistTw, currentDuration);
                    if (result) {
                        console.log(`[Lyrics] Found from ${source.name} (converted):`, titleTw, artistTw);
                        this.lrcData = result.lrcData;
                        this.updateCache(title, artist, result.lyrics);
                        fetchRetryCount = 0;
                        return;
                    }
                }
            } catch (e) {
                lastError = e;
                console.warn(`[Lyrics] ${source.name} failed:`, e);
            }
        }

        fetchRetryCount++;
        console.warn(`[Lyrics] All sources failed (${fetchRetryCount}/${MAX_RETRY}):`, lastError);
    }

    async updateCache(title, artist, lrc) {
        try {
            await fetch(`${API_BASE}/update_cache`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title, artist, lrc })
            });
        } catch (e) { /* fail silently */ }
    }

    startRenderLoop() {
        const loop = () => {
            this.computeFrameData();
            this.onFrame?.(this.frameData);
            requestAnimationFrame(loop);
        };
        requestAnimationFrame(loop);
    }

    computeFrameData() {
        const pos = this.lastPosition;
        const lines = this.lrcData;

        if (!lines.length) {
            this.frameData = { currentIndex: 0, lineProgress: 0, isInterlude: false, countdown: 0, velocity: 0 };
            return;
        }

        let idx = 0;
        for (let i = 0; i < lines.length - 1; i++) {
            if (pos >= lines[i].time && pos < lines[i + 1].time) {
                idx = i;
                break;
            }
            if (pos >= lines[lines.length - 1].time) idx = lines.length - 1;
        }

        const curr = lines[idx];
        const next = lines[idx + 1];
        const duration = next ? next.time - curr.time : (curr.time + 5000);
        const elapsed = pos - curr.time;
        const progress = Math.max(0, Math.min(1, elapsed / duration));

        const isInterlude = this.detectInterlude(curr.text);
        const countdown = isInterlude ? Math.max(0, Math.ceil((curr.time - pos) / 1000)) : 0;

        const deltaPos = pos - (this._prevPosition || pos);
        this.velocity = Math.abs(deltaPos);
        this._prevPosition = pos;
        this.frameData = {
            currentIndex: idx,
            lineProgress: progress,
            isInterlude,
            countdown: isInterlude ? Math.max(0, Math.ceil((curr.time - pos) / 1000)) : 0,
            velocity: this.velocity
        };
    }

    detectInterlude(text) {
        const interludeKeywords = ['间奏', '器乐', 'instrumental', 'interlude', '空白', '...'];
        return interludeKeywords.some(k => text.toLowerCase().includes(k.toLowerCase()));
    }
}

// 修复变量名冲突
const motionEngine = new MotionEngine();
window.motion = motionEngine;

// 重试计数
let fetchRetryCount = 0;
const MAX_RETRY = 3;