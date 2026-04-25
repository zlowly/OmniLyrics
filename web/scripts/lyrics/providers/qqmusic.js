// qqmusic.js - QQ音乐歌词源
// 功能：通过后端代理搜索歌曲 → 获取加密歌词 → 后端解密

const QQ_MUSIC_API_BASE = (() => {
    const loc = window.location;
    return `${loc.protocol}//${loc.host}`;
})();

class QQMusicProvider extends window.LyricsProvider {
    name = 'qqmusic';

    async search(title, artist, duration) {
        try {
            const songMid = await this.searchSong(title, artist);
            if (!songMid) return null;

            const encrypted = await this.getEncryptedLyrics(songMid);
            if (!encrypted) return null;

            const lyrics = await this.decrypt(encrypted);
            if (!lyrics) return null;

            const lrcData = window.motion?.parseLRC(lyrics) || [];
            return { lrcData, lyrics };
        } catch (e) {
            console.warn(`[QQMusic] Search failed:`, e);
            return null;
        }
    }

    async searchSong(title, artist) {
        const resp = await fetch(`${QQ_MUSIC_API_BASE}/proxy/qqmusic/search`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ title, artist })
        });

        if (!resp.ok) {
            console.warn('[QQMusic] Search failed:', resp.status);
            return null;
        }

        const json = await resp.json();
        if (json.error) {
            console.warn('[QQMusic] Search error:', json.error);
            return null;
        }

        return json.songMid || null;
    }

    async getEncryptedLyrics(songMid) {
        const resp = await fetch(`${QQ_MUSIC_API_BASE}/proxy/qqmusic/lyric`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ songMid })
        });

        if (!resp.ok) {
            console.warn('[QQMusic] Lyric fetch failed:', resp.status);
            return null;
        }

        const json = await resp.json();
        if (json.error) {
            console.warn('[QQMusic] Lyric error:', json.error);
            return null;
        }

        return json.encrypted || null;
    }

    async decrypt(encryptedHex) {
        const resp = await fetch(`${QQ_MUSIC_API_BASE}/decrypt`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ encrypted: encryptedHex })
        });

        if (!resp.ok) return null;

        const json = await resp.json();
        return json.lyrics || null;
    }
}

window.QQMusicProvider = QQMusicProvider;