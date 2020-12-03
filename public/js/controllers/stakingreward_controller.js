import { Controller } from 'stimulus'

export default class extends Controller {
  static get targets () {
    return [
      'blockHeight', 'ticketPrice', 'ticketReward', 'rewardPeriod',
      'priceDCR', 'dayText', 'amount', 'amountText', 'days', 'daysText',
      'amountRoi', 'percentageRoi', 'tickets', 'amountUsd', 'amountRoiUsd'
    ]
  }

  async connect () {
    this.height = parseInt(this.data.get('height'))
    this.ticketPrice = parseInt(this.data.get('ticketPrice'))
    this.dcrPrice = parseFloat(this.data.get('dcrprice'))
    this.ticketReward = parseFloat(this.data.get('ticketReward'))
    this.rewardPeriod = parseFloat(this.data.get('rewardPeriod'))

    this.blockHeightTarget.textContent = this.height
    this.ticketPriceTarget.textContent = this.ticketPrice
    this.ticketRewardTarget.textContent = this.ticketReward.toFixed(2)
    this.rewardPeriodTarget.textContent = this.rewardPeriod.toFixed(2)

    this.priceDCRTarget.value = this.dcrPrice.toFixed(2)
  }

  updatePrice () {
    this.dcrPrice = parseInt(this.dcrPriceTarget.value)
  }

  calculate () {
    const amount = parseFloat(this.amountTarget.value)
    const days = parseFloat(this.daysTarget.value)
    if (days < this.rewardPeriod) {
      window.alert(`You must stake for ${this.rewardPeriod} days and above`)
      return
    }
    this.ticketsTarget.textContent = parseInt(amount / this.ticketPrice)
    this.amountTextTarget.textContent = amount
    this.amountUsdTarget.textContent = (amount * this.dcrPrice).toFixed(2)
    this.daysTextTarget.textContent = days

    // number of periods
    const periods = parseInt(days / this.rewardPeriod)
    const totalPercentage = periods * this.ticketReward
    const totalAmount = totalPercentage * amount * 1 / 100
    this.percentageRoiTarget.textContent = totalPercentage.toFixed(2)
    this.amountRoiTarget.textContent = totalAmount.toFixed(2)
    this.amountRoiUsdTarget.textContent = (totalAmount * this.dcrPrice).toFixed(2)
  }
}
