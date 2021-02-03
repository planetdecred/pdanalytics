// For all your value formatting needs...

var humanize = {
  timeSince: function (unixTime, keepOnly) {
    var seconds = Math.floor(((new Date().getTime() / 1000) - unixTime))
    var interval = Math.floor(seconds / 31536000)
    if (interval >= 1) {
      let extra = Math.floor((seconds - interval * 31536000) / 2628000)
      let result = interval + 'y'
      if (extra > 0 && keepOnly !== 'years') {
        result = result + ' ' + extra + 'mo'
      }
      return result
    }
    interval = Math.floor(seconds / 2628000)
    if (interval >= 1) {
      let extra = Math.floor((seconds - interval * 2628000) / 86400)
      let result = interval + 'mo'
      if (extra > 0 && keepOnly !== 'months') {
        result = result + ' ' + extra + 'd'
      }
      return result
    }
    interval = Math.floor(seconds / 86400)
    if (interval >= 1) {
      let extra = Math.floor((seconds - interval * 86400) / 3600)
      let result = interval + 'd'
      if (extra > 0 && keepOnly !== 'days') {
        result = result + ' ' + extra + 'h'
      }
      return result
    }
    interval = Math.floor(seconds / 3600)
    if (interval >= 1) {
      let extra = Math.floor((seconds - interval * 3600) / 60)
      let result = interval + 'h'
      if (extra > 0) {
        result = result + ' ' + extra + 'm'
      }
      return result
    }
    interval = Math.floor(seconds / 60)
    if (interval >= 1) {
      let extra = seconds - interval * 60
      let result = interval + 'm'
      if (extra > 0) {
        result = result + ' ' + extra + 's'
      }
      return result
    }
    return Math.floor(seconds) + 's'
  },
  date: function (stamp, withTimezone, hideTime) {
    var d = new Date(stamp)
    var dateStr = `${String(d.getUTCFullYear())}-${String(d.getUTCMonth() + 1).padStart(2, '0')}-${String(d.getUTCDate()).padStart(2, '0')}`
    const isMidnight = d.getUTCHours() === 0 && d.getUTCMinutes() === 0 && d.getUTCSeconds() === 0
    if (hideTime || isMidnight) {
      if (withTimezone) dateStr += ' (UTC)'
      return dateStr
    }
    dateStr += ` ${String(d.getUTCHours()).padStart(2, '0')}:${String(d.getUTCMinutes()).padStart(2, '0')}:${String(d.getUTCSeconds()).padStart(2, '0')}`
    if (withTimezone) dateStr += ' (UTC)'
    return dateStr
  }
}

export default humanize
