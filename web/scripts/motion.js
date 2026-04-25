// motion.js - 驱动层：逻辑运动引擎
// 职责：心跳监测、多源竞态调度、状态计算

// 从当前页面 URL 自动提取主机和端口，用于 API 调用
const API_BASE = (() => {
    const loc = window.location;
    return `${loc.protocol}//${loc.host}`;
})();
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
    console.log('[Lrclib] Searching:', title, artist, 'duration:', currentDuration);
    console.log('[Lrclib] URL:', searchUrl);
    
    const res = await fetch(searchUrl);
    console.log('[Lrclib] Search response status:', res.status);
    if (!res.ok) return null;
    
    const results = await res.json();
    console.log('[Lrclib] Search results count:', results?.length);
    if (!Array.isArray(results) || results.length === 0) return null;

    const withDuration = results.filter(r => r.duration && r.duration > 0);
    console.log('[Lrclib] Results with duration:', withDuration?.length);
    const best = withDuration.length === 0
        ? results[0]
        : withDuration.sort((a, b) => Math.abs(a.duration - currentDuration) - Math.abs(b.duration - currentDuration))[0];
    
    console.log('[Lrclib] Best match:', best?.track_name, 'duration:', best?.duration);

    const getUrl = `${LRCLIB_API.replace('/get', '/get')}/${best.id}`;
    const getRes = await fetch(getUrl);
    console.log('[Lrclib] Get response status:', getRes.status);
    if (!getRes.ok) return null;
    
    const json = await getRes.json();
    console.log('[Lrclib] Got lyrics, has synced:', !!json?.syncedLyrics, 'has plain:', !!json?.plainLyrics);
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
    const lineTimeRegex = /\[(\d{2}):(\d{2})\.(\d{2,3})\]/;
    const cleaned = cleanLRCInternal(lrcText);

    const rawLines = cleaned.split('\n');
    for (const rawLine of rawLines) {
        const trimmed = rawLine.trim();
        if (!trimmed) continue;

        // 查找行首的第一个时间戳作为行时间
        const lineMatch = lineTimeRegex.exec(trimmed);
        if (!lineMatch) continue;

        const min = parseInt(lineMatch[1]);
        const sec = parseInt(lineMatch[2]);
        const ms = lineMatch[3].length === 3 ? parseInt(lineMatch[3]) : parseInt(lineMatch[3]) * 10;
        const lineTimeMs = (min * 60 + sec) * 1000 + ms;

        // 提取行文本（去除行首时间戳）
        const text = trimmed.substring(lineMatch[0].length);

        // 解析逐字时间戳：每个字前面都有 [mm:ss.xx]
        // 格式：[00:00.000]汪[00:01.054]苏[00:02.108]泷
        const words = [];
        const wordRegex = /\[(\d{2}):(\d{2})\.(\d{2,3})\]([^\[]+)/g;
        let match;

        console.log('[LRC] Parsing line:', trimmed);

        while ((match = wordRegex.exec(trimmed)) !== null) {
            const wMin = parseInt(match[1]);
            const wSec = parseInt(match[2]);
            const wMs = match[3].length === 3 ? parseInt(match[3]) : parseInt(match[3]) * 10;
            const wordTimeMs = (wMin * 60 + wSec) * 1000 + wMs;
            const wordText = match[4]; // 时间戳后面的文本

            console.log('[LRC] Word match:', match[0], '-> time:', wordTimeMs, 'text:', wordText);

            // 逐字拆分（支持汉字和英文字符）
            for (const char of wordText) {
                if (char.trim()) {
                    words.push({ time: wordTimeMs, text: char });
                }
            }
        }

        console.log('[LRC] Words found:', words.length);

        // 如果有逐字时间戳（words 数量 > 1），使用逐字格式
        // 同时清理 text 中的时间戳，只保留纯文本
        const cleanText = text.replace(/\[\d{2}:\d{2}\.\d{2,3}\]/g, '').trim();

        if (words.length > 1) {
            lines.push({ time: lineTimeMs, text: cleanText, words: words });
        } else if (cleanText) {
            // 普通行
            lines.push({ time: lineTimeMs, text: cleanText });
        }
    }

    // 计算每行的结束时间（下一行的开始时间）
    const sorted = lines.sort((a, b) => a.time - b.time);
    for (let i = 0; i < sorted.length - 1; i++) {
        sorted[i].endTime = sorted[i + 1].time;
    }
    // 最后一行的结束时间默认为开始时间 + 2 秒
    if (sorted.length > 0) {
        sorted[sorted.length - 1].endTime = sorted[sorted.length - 1].time + 2000;
    }

    return sorted;
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

const lyricsSources = window.lyricsSources || [];
let scheduler = null;

async function getScheduler() {
    if (!scheduler) {
        scheduler = new window.LyricsScheduler();
        scheduler.init(window.lyricsSources || []);
    }
    return scheduler;
}

class MotionEngine {
    constructor() {
        this.isConnected = false;
        this.lrcData = [];
        this.currentIndex = 0;
        this.lastPosition = 0;
        this.lastTitle = '';
        this.songDuration = 0;  // 歌曲总时长
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
        this.songDuration = data.duration || 0;
        const duration = this.songDuration;
        this.lastStatus = data.status || 'Unknown';
        this.lastAppName = data.appName || '';

        // 检测换歌，重置重试计数和歌词
        if (data.title !== this.lastTitle) {
            this.lastTitle = data.title;
            this.lrcData = [];
            fetchRetryCount = 0;
            this.fetchFailed = false;
            console.log('[Motion] New song:', data.title, 'duration:', duration);

            // 通知渲染器重置状态
            this.onSongChange?.();

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
        if (!currentDuration || currentDuration <= 0) {
            console.warn('[Lyrics] No duration, skip search');
            return;
        }
        if (fetchRetryCount >= MAX_RETRY) {
            console.warn('[Lyrics] Max retries reached, giving up');
            this.fetchFailed = true;
            return;
        }

        const appName = this.lastAppName || '';
        const sched = await getScheduler();

        let result = await sched.search(title, artist, currentDuration, appName);

        if (result) {
            console.log('[Lyrics] Found:', title, artist);
            this.lrcData = result.lrcData;
            this.updateCache(title, artist, result.lyrics);
            fetchRetryCount = 0;
            return;
        }

        const titleTw = await convertToTraditional(title);
        const artistTw = await convertToTraditional(artist);
        if (titleTw !== title || artistTw !== artist) {
            result = await sched.search(titleTw, artistTw, currentDuration, appName);
            if (result) {
                console.log('[Lyrics] Found (converted):', titleTw, artistTw);
                this.lrcData = result.lrcData;
                this.updateCache(title, artist, result.lyrics);
                fetchRetryCount = 0;
                return;
            }
        }

        fetchRetryCount++;
        console.warn(`[Lyrics] All sources failed (${fetchRetryCount}/${MAX_RETRY})`);
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

        // 检测播放结束
        const isSongEnded = this.songDuration > 0 && pos >= this.songDuration;

        this.frameData = {
            currentIndex: idx,
            lineProgress: progress,
            isInterlude,
            countdown: isInterlude ? Math.max(0, Math.ceil((curr.time - pos) / 1000)) : 0,
            velocity: this.velocity,
            position: pos,
            isPlaying: this.lastStatus === 'Playing',
            isSongEnded
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

// 歌词 provider 定义
class LrclibProvider {
    name = 'lrclib';
    async search(title, artist, duration) {
        return searchLrclib(title, artist, duration);
    }
}

// 注册到全局，LyricsScheduler 初始化时会读取
// QQMusicProvider 来自 qqmusic.js（已在 index.html 中加载）
const providers = [new LrclibProvider()];
if (window.QQMusicProvider) {
    providers.push(new window.QQMusicProvider());
}
window.lyricsSources = providers;

// 重试计数
let fetchRetryCount = 0;
const MAX_RETRY = 3;