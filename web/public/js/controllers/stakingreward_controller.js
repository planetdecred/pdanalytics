import { Controller } from 'stimulus'
import axios from 'axios'
import moment from 'moment'

export default class extends Controller {
  static get targets () {
    return [
      'blockHeight', 'ticketPrice',
      'startDate', 'endDate',
      'priceDCR', 'dayText', 'amount', 'amountText', 'days', 'daysText',
      'amountRoi', 'percentageRoi', 'tickets'
    ]
  }

  async connect () {
    this.ticketPrice = parseFloat(this.data.get('ticketPrice'))
  }

  calculate () {
    const _this = this
    const amount = parseFloat(this.amountTarget.value)
    if (!(amount > 0)) {
      window.alert('Amount must be greater than 0')
      return
    }
    let startDate = moment(this.startDateTarget.value)
    let endDate = moment(this.endDateTarget.value)

    const days = moment.duration(endDate.diff(startDate)).asDays()
    if (days < this.rewardPeriod) {
      window.alert(`You must stake for ${this.rewardPeriod.toFixed(2)} days and above`)
      return
    }

    let startDateUnix = new Date(this.startDateTarget.value).getTime()
    let endDateUnix = new Date(this.endDateTarget.value).getTime()
    let url = `/staking-reward/get-future-reward?startDate=${startDateUnix}&endDate=${endDateUnix}&startingBalance=${amount}`
    axios.get(url).then(function (response) {
      let result = response.data

      _this.ticketsTarget.textContent = parseInt(amount / result.ticketPrice)
      _this.amountTextTarget.textContent = amount
      _this.daysTextTarget.textContent = days

      // number of periods
      const totalPercentage = result.reward
      const totalAmount = totalPercentage * amount * 1 / 100
      _this.percentageRoiTarget.textContent = totalPercentage.toFixed(2)
      _this.amountRoiTarget.textContent = totalAmount.toFixed(2)
    })
  }
}
