// motion.js - 驱动层：逻辑运动引擎
// 简化版：直接调用后端 /lyrics 接口获取歌词

// 从当前页面 URL 自动提取主机和端口，用于 API 调用
const API_BASE = (() => {
    const loc = window.location;
    return `${loc.protocol}//${loc.host}`;
})();

// 解析 LRC 格式歌词为结构化数据
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
        const words = [];
        const wordRegex = /\[(\d{2}):(\d{2})\.(\d{2,3})\]([^\[]+)/g;
        let match;

        while ((match = wordRegex.exec(trimmed)) !== null) {
            const wMin = parseInt(match[1]);
            const wSec = parseInt(match[2]);
            const wMs = match[3].length === 3 ? parseInt(match[3]) : parseInt(match[3]) * 10;
            const wordTimeMs = (wMin * 60 + wSec) * 1000 + wMs;
            const wordText = match[4];

            // 逐字拆分（支持汉字和英文字符）
            for (const char of wordText) {
                if (char.trim()) {
                    words.push({ time: wordTimeMs, text: char });
                }
            }
        }

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

class MotionEngine {
    constructor() {
        this.isConnected = false;
        this.lrcData = [];
        this.currentIndex = 0;
        this.lastPosition = 0;
        this.lastTitle = '';
        this.songDuration = 0;
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
        this.onSongChange = null;
        this.online = navigator.onLine;
        this.fetchFailed = false;
        this.fetchRetryCount = 0;
        this.lastAppName = '';
        this.lastStatus = 'Unknown';
        this.lastDuration = 0; // 新增：记录上一次的duration
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
        }, 250);
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
        
        // 检测应用名称是否变化（此时this.lastAppName仍是上一次的值）
        const appNameChanged = data.appName !== this.lastAppName;
        
        // 更新应用名称
        this.lastAppName = data.appName || '';

        // 检测切歌：标题变化 或 应用变化 或 duration变化（且duration>0）
        const songChanged = data.title !== this.lastTitle || 
                           appNameChanged || 
                           (duration > 0 && duration !== this.lastDuration);
        
        if (songChanged && data.title) {
            this.lastTitle = data.title;
            this.lastDuration = duration; // 记录新的duration
            this.lrcData = [];
            this.fetchRetryCount = 0;
            this.fetchFailed = false;
            console.log('[Motion] New song:', data.title, 'duration:', duration);

            // 通知渲染器重置状态
            this.onSongChange?.();

            // 获取歌词（duration可能仍为0，fetchLyrics内部会处理）
            await this.fetchLyrics(data.title, data.artist, Math.floor(duration / 1000), this.lastAppName);
        } else if (!songChanged) {
            // 未切歌，更新lastDuration（用于下次比较）
            this.lastDuration = duration;
        }

        this.lastPosition = position;
    }

    async fetchLyrics(title, artist, durationSec, appName) {
        if (!title || !this.online) return;
        if (!durationSec || durationSec <= 0) {
            console.warn('[Lyrics] No duration, skip search');
            return;
        }
        if (this.fetchRetryCount >= 3) {
            console.warn('[Lyrics] Max retries reached, giving up');
            this.fetchFailed = true;
            return;
        }

        try {
            const url = `${API_BASE}/lyrics?title=${encodeURIComponent(title)}&artist=${encodeURIComponent(artist || '')}&duration=${durationSec}&appName=${encodeURIComponent(appName || '')}`;
            console.log('[Lyrics] Fetching from backend:', url);

            const res = await fetch(url);
            if (!res.ok) {
                throw new Error(`HTTP ${res.status}`);
            }

            const result = await res.json();
            console.log('[Lyrics] Backend response:', result);

            if (result.found && result.lyrics) {
                console.log('[Lyrics] Found lyrics from', result.source, '(cached:', result.cached, ')');
                this.lrcData = parseLRCInternal(result.lyrics);
                console.log('[Motion] Parsed lrcData length:', this.lrcData.length);
                this.fetchRetryCount = 0;
                return;
            }

            // 未找到歌词，尝试转换为繁体再搜索
            const titleTw = await this.convertToTraditional(title);
            const artistTw = await this.convertToTraditional(artist);
            if (titleTw !== title || artistTw !== artist) {
                const retryUrl = `${API_BASE}/lyrics?title=${encodeURIComponent(titleTw)}&artist=${encodeURIComponent(artistTw || '')}&duration=${durationSec}&appName=${encodeURIComponent(appName || '')}`;
                console.log('[Lyrics] Retry with traditional Chinese:', retryUrl);

                const retryRes = await fetch(retryUrl);
                if (retryRes.ok) {
                    const retryResult = await retryRes.json();
                    if (retryResult.found && retryResult.lyrics) {
                        console.log('[Lyrics] Found lyrics (traditional):', retryResult.source);
                        this.lrcData = parseLRCInternal(retryResult.lyrics);
                        this.fetchRetryCount = 0;
                        return;
                    }
                }
            }

            this.fetchRetryCount++;
            console.warn(`[Lyrics] Not found (${this.fetchRetryCount}/3)`);
        } catch (e) {
            console.warn('[Lyrics] Fetch failed:', e);
            this.fetchRetryCount++;
        }
    }

    async convertToTraditional(text) {
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

    parseLRC(lrcText) {
        return parseLRCInternal(lrcText);
    }

    cleanLRC(lrc) {
        return cleanLRCInternal(lrc);
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

// 导出实例
const motionEngine = new MotionEngine();
window.motion = motionEngine;
