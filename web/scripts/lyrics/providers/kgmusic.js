// kgmusic.js - KGMusic歌词源前端提供者
class KGMusicProvider extends window.LyricsProvider {
    name = 'kugou';

    async search(title, artist, duration) {
        try {
            const songInfo = await this.searchSong(title, artist);
            if (!songInfo) return null;
            const lyricInfo = await this.getEncryptedLyrics(songInfo);
            if (!lyricInfo) return null;
            const lyrics = await this.decrypt(lyricInfo.encrypted);
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
        return data;
    }

    async getEncryptedLyrics(songInfo) {
        const resp = await fetch('/proxy/kgmusic/lyric', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ songId: songInfo.songId || songInfo.SongID, hash: songInfo.hash, duration: songInfo.duration })
        });
        if (!resp.ok) return null;
        return await resp.json();
    }

    async decrypt(encrypted) {
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
