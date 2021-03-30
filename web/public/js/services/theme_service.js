import { setCookie } from './cookie_service'
import globalEventBus from './event_bus_service'

var darkBGCookieName = 'dcrdataDarkBG'

export function darkEnabled () {
  return document.cookie.includes(darkBGCookieName)
}

function menuToggle () {
  return document.querySelector('#menuToggle input')
}

if (darkEnabled()) {
  toggleToDarkClasses(document.body)
} else {
  toggleToLightClasses(document.body)
}
function toggleToDarkClasses (body) {
  body.classList.add('darkBG')
}
function toggleToLightClasses (body) {
  body.classList.remove('darkBG')
}
export function toggleSun () {
  if (darkEnabled()) {
    setCookie(darkBGCookieName, '', 0)
    toggleToLightClasses(document.body)
    globalEventBus.publish('NIGHT_MODE', { nightMode: false })
  } else {
    setCookie(darkBGCookieName, 1, 525600)
    toggleToDarkClasses(document.body)
    globalEventBus.publish('NIGHT_MODE', { nightMode: true })
  }
}

document.addEventListener('turbolinks:before-render', function (event) {
  if (darkEnabled()) {
    toggleToDarkClasses(event.data.newBody)
  } else {
    toggleToLightClasses(event.data.newBody)
  }
})

export function toggleMenu () {
  var checkbox = menuToggle()
  checkbox.checked = !checkbox.checked
  checkbox.dispatchEvent(new window.Event('change'))
}

export function closeMenu () {
  var checkbox = menuToggle()
  if (!checkbox.checked) return
  checkbox.checked = false
  checkbox.dispatchEvent(new window.Event('change'))
}
