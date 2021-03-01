import { Controller } from 'stimulus'

export default class extends Controller {
  navbarIsOpened

  static get targets () {
    return [
      'navbar'
    ]
  }

  toggleNavbar () {
    if (!this.navbarIsOpened) {
      $(this.navbarTarget).slideDown()
    } else {
      $(this.navbarTarget).slideUp()
    }
    this.navbarIsOpened = !this.navbarIsOpened
  }
}
