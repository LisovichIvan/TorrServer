import axios from 'axios'
import parseTorrent from 'parse-torrent'
import ptt from 'parse-torrent-title'

export const getMoviePosters = (movieName, language = 'en') => {
  const url = 'http://api.themoviedb.org/3/search/multi'

  return axios
    .get(url, {
      params: {
        api_key: process.env.REACT_APP_TMDB_API_KEY,
        language,
        include_image_language: `${language},null,en`,
        query: movieName,
      },
    })
    .then(({ data: { results } }) =>
      results.filter(el => el.poster_path).map(el => `https://image.tmdb.org/t/p/w300${el.poster_path}`),
    )
    .catch(() => null)
}

export const checkImageURL = async url => {
  if (!url || !url.match(/.(jpg|jpeg|png|gif)$/i)) return false

  try {
    await fetch(url)
    return true
  } catch (e) {
    return false
  }
}

const magnetRegex = /^magnet:\?xt=urn:[a-z0-9].*/i
export const hashRegex = /^\b[0-9a-f]{32}\b$|^\b[0-9a-f]{40}\b$|^\b[0-9a-f]{64}\b$/i
const torrentRegex = /^.*\.(torrent)$/i
export const chechTorrentSource = source =>
  source.match(hashRegex) !== null || source.match(magnetRegex) !== null || source.match(torrentRegex) !== null

export const parseTorrentTitle = (parsingSource, callback) => {
  parseTorrent.remote(parsingSource, (err, { name, files } = {}) => {
    if (!name || err) return callback({ parsedTitle: null, originalName: null })

    const torrentName = ptt.parse(name).title
    const nameOfFileInsideTorrent = files ? ptt.parse(files[0].name).title : null

    let newTitle = torrentName
    if (nameOfFileInsideTorrent) {
      // taking shorter title because in most cases it is more accurate
      newTitle = torrentName.length < nameOfFileInsideTorrent.length ? torrentName : nameOfFileInsideTorrent
    }

    callback({ parsedTitle: newTitle, originalName: name })
  })
}
