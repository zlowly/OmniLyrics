// qqmusic.js - QQ音乐歌词源
// 功能：搜索歌曲 → 获取QRC加密歌词 → 后端解密

const QQ_MUSIC_API = 'https://u.y.qq.com/cgi-bin/musicu.fcg';

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
        const query = artist ? `${artist} ${title}` : title;
        const data = {
            "req_1": {
                "method": "DoSearchForQQMusicDesktop",
                "module": "music.search.SearchCgiService",
                "param": {
                    "num_per_page": "10",
                    "page_num": "1",
                    "query": query,
                    "search_type": 0
                }
            }
        };

        const resp = await fetch(QQ_MUSIC_API, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Referer': 'https://c.y.qq.com/'
            },
            body: JSON.stringify(data)
        });

        const json = await resp.json();
        const songs = json?.req_1?.data?.song?.list;
        if (!songs || songs.length === 0) return null;

        const best = this.findBestMatch(songs, duration);
        return best?.songmid || songs[0]?.songmid;
    }

    findBestMatch(songs, targetDuration) {
        if (!targetDuration || !songs) return null;
        const withDuration = songs.filter(s => s.duration > 0);
        if (withDuration.length === 0) return null;

        return withDuration.reduce((best, song) => {
            const diff = Math.abs(song.duration - targetDuration);
            const bestDiff = best ? Math.abs(best.duration - targetDuration) : Infinity;
            return diff < bestDiff ? song : best;
        }, null);
    }

    async getEncryptedLyrics(songMid) {
        const currentMillis = Date.now();
        const data = new URLSearchParams({
            'callback': 'MusicJsonCallback_lrc',
            'pcachetime': currentMillis.toString(),
            'songmid': songMid,
            'g_tk': '5381',
            'jsonpCallback': 'MusicJsonCallback_lrc',
            'loginUin': '0',
            'hostUin': '0',
            'format': 'jsonp',
            'inCharset': 'utf8',
            'outCharset': 'utf8',
            'notice': '0',
            'platform': 'yqq',
            'needNewCode': '0'
        });

        const resp = await fetch(
            `https://c.y.qq.com/lyric/fcgi-bin/fcg_query_lyric_new.fcg?${data}`,
            { headers: { 'Referer': 'https://c.y.qq.com/' } }
        );

        const text = await resp.text();
        const json = this.parseJSONP(text, 'MusicJsonCallback_lrc');
        return json?.lyric || null;
    }

    parseJSONP(text, callback) {
        const prefix = callback + '(';
        if (!text.startsWith(prefix)) return null;
        const jsonStr = text.slice(prefix.length, -1);
        try {
            return JSON.parse(jsonStr);
        } catch {
            return null;
        }
    }

    async decrypt(encryptedHex) {
        const API_BASE = window.location.origin;

        const resp = await fetch(`${API_BASE}/decrypt`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ encrypted: encryptedHex })
        });

        if (!resp.ok) return null;
        const json = await resp.json();
        return json?.lyrics || null;
    }
}

window.QQMusicProvider = QQMusicProvider;