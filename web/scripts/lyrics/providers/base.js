// base.js - 歌词源接口定义
// 所有歌词源需实现此接口

class LyricsProvider {
    name = 'unknown';

    /**
     * 搜索歌词
     * @param {string} title - 歌曲名
     * @param {string} artist - 艺术家
     * @param {number} duration - 时长(秒)
     * @returns {Promise<{lrcData: Array, lyrics: string} | null>}
     */
    async search(title, artist, duration) {
        throw new Error('Not implemented');
    }
}

window.LyricsProvider = LyricsProvider;