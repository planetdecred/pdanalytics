import { Controller } from 'stimulus'
import axios from 'axios'
import moment from 'moment'

export default class extends Controller {
  static get targets () {
    return [
      'blockHeight', 'ticketPrice', 'ticketReward', 'rewardPeriod',
      'startDate', 'endDate',
      'priceDCR', 'dayText', 'amount', 'amountText', 'days', 'daysText',
      'amountRoi', 'percentageRoi', 'tickets', 'amountUsd', 'amountRoiUsd'
    ]
  }

  async connect () {
    this.height = parseInt(this.data.get('height'))
    this.ticketPrice = parseFloat(this.data.get('ticketPrice'))
    this.dcrPrice = parseFloat(this.data.get('dcrprice'))
    this.ticketReward = parseFloat(this.data.get('ticketReward'))
    this.rewardPeriod = parseFloat(this.data.get('rewardPeriod'))

    this.blockHeightTarget.textContent = this.height
    this.ticketPriceTarget.textContent = this.ticketPrice.toFixed(2)
    this.ticketRewardTarget.textContent = this.ticketReward.toFixed(2)
    this.rewardPeriodTarget.textContent = this.rewardPeriod.toFixed(2)

    this.priceDCRTarget.value = this.dcrPrice.toFixed(2)
  }

  updatePrice () {
    this.dcrPrice = parseInt(this.dcrPriceTarget.value)
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
      _this.amountUsdTarget.textContent = (amount * _this.dcrPrice).toFixed(2)
      _this.daysTextTarget.textContent = days

      // number of periods
      const totalPercentage = result.reward
      const totalAmount = totalPercentage * amount * 1 / 100
      _this.percentageRoiTarget.textContent = totalPercentage.toFixed(2)
      _this.amountRoiTarget.textContent = totalAmount.toFixed(2)
      _this.amountRoiUsdTarget.textContent = (totalAmount * _this.dcrPrice).toFixed(2)
    })
  }
}
