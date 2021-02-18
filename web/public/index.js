import 'regenerator-runtime/runtime'
/* global require */
import { Application } from 'stimulus'
import { definitionsFromContext } from 'stimulus/webpack-helpers'

require('./scss/application.scss')

const application = Application.start()
const context = require.context('./js/controllers', true, /\.js$/)
application.load(definitionsFromContext(context))

document.addEventListener('turbolinks:load', function (e) {
  document.querySelectorAll('.jsonly').forEach((el) => {
    el.classList.remove('jsonly')
  })
})

// Debug logging can be enabled by entering logDebug(true) in the console.
// Your setting will persist across sessions.
window.loggingDebug = window.localStorage.getItem('loggingDebug') === '1'
window.logDebug = yes => {
  window.loggingDebug = yes
  window.localStorage.setItem('loggingDebug', yes ? '1' : '0')
  return 'debug logging set to ' + (yes ? 'true' : 'false')
}
