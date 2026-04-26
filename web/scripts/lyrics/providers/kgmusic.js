// kgmusic.js - KGMusic歌词源前端提供者
class KGMusicProvider extends window.LyricsProvider {
    name = 'kgmusic';

    async search(title, artist, duration) {
        try {
            const songInfo = await this.searchSong(title, artist);
            if (!songInfo) return null;
            const lyricInfo = await this.getEncryptedLyrics(songInfo);
            if (!lyricInfo) return null;

            // 如果已有 lyrics（在 encrypted 为空时获取的 LRC），直接使用
            if (lyricInfo.lyrics) {
                const lrcData = window.motion?.parseLRC(lyricInfo.lyrics) || [];
                return { lrcData, lyrics: lyricInfo.lyrics };
            }

            // 否则尝试解密 KRC 格式
            const lyrics = await this.decrypt(lyricInfo.encrypted);
            if (!lyrics) return null;
            const lrcData = window.motion?.parseLRC(lyrics) || [];
            return { lrcData, lyrics };
        } catch (e) {
            console.warn('[KGMusic] Search failed:', e);
            return null;
        }
    }

    async searchSong(title, artist) {
        const resp = await fetch('/proxy/kgmusic/search', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ title, artist })
        });
        if (!resp.ok) return null;
        const data = await resp.json();
        console.log('[KGMusic] search response', data);
        return data;
    }

    async getEncryptedLyrics(songInfo) {
        const resp = await fetch('/proxy/kgmusic/lyric', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ songId: songInfo.songId || songInfo.SongID, hash: songInfo.hash, duration: songInfo.duration })
        });
        if (!resp.ok) { console.warn('[KGMusic] lyric fetch failed', resp.status); return null; }
        const data = await resp.json();
        console.log('[KGMusic] lyric response', data);

        // 如果响应中已包含 lyrics（encrypted 为空时的后备），直接返回
        if (data.lyrics) {
            return { lyrics: data.lyrics };
        }
        return data;
    }

    async decrypt(encrypted) {
        // 如果已经有 lyrics（在 getEncryptedLyrics 中已获取），直接返回
        if (encrypted && encrypted.lyrics) {
            return encrypted.lyrics;
        }
        const resp = await fetch('/decrypt-krc', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ encrypted })
        });
        if (!resp.ok) return '';
        const data = await resp.json();
        return data.lyrics || '';
    }
}

window.KGMusicProvider = KGMusicProvider;
