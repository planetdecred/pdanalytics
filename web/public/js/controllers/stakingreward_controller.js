import { Controller } from 'stimulus'
import axios from 'axios'
import moment from 'moment'
import TurboQuery from '../helpers/turbolinks_helper'
import { hide, insertOrUpdateQueryParam, show } from '../utils'

export default class extends Controller {
  static get targets () {
    return [
      'blockHeight', 'ticketPrice',
      'startDate', 'endDate',
      'priceDCR', 'dayText', 'amount', 'days', 'daysText',
      'amountRoi', 'percentageRoi',
      'table', 'tableBody', 'rowTemplate'
    ]
  }

  async initialize () {
    this.ticketPrice = parseFloat(this.data.get('ticketPrice'))
    this.rewardPeriod = parseInt(this.data.get('rewardPeriod'))

    this.lastYear = moment().subtract(1, 'year')
    this.startDateTarget.value = this.lastYear.format('YYYY-MM-DD')
    this.now = moment()
    this.endDateTarget.value = this.now.format('YYYY-MM-DD')
    this.amountTarget.value = 1000

    this.query = new TurboQuery()
    this.settings = TurboQuery.nullTemplate([
      'amount', 'start', 'end'
    ])
    this.query.update(this.settings)
    if (this.settings.amount) {
      this.amountTarget.value = this.settings.amount
    }
    if (this.settings.start) {
      this.startDateTarget.value = moment(this.settings.start).format('YYYY-MM-DD')
    }
    if (this.settings.end) {
      this.endDateTarget.value = moment(this.settings.end).format('YYYY-MM-DD')
    }

    this.calculate()
  }

  amountKeypress (e) {
    console.log(e.keyCode)
    console.log(e)
    if (e.keyCode === 13) {
      this.amountChanged()
    }
  }

  amountChanged () {
    insertOrUpdateQueryParam('amount', parseInt(this.amountTarget.value), 1000)
    this.calculate()
  }

  startDateChanged () {
    let startDateUnix = new Date(this.startDateTarget.value).getTime()
    insertOrUpdateQueryParam('start', startDateUnix, parseInt(this.lastYear.format('X')))
    this.calculate()
  }

  endDateChanged () {
    let endDateUnix = new Date(this.endDateTarget.value).getTime()
    insertOrUpdateQueryParam('end', endDateUnix, parseInt(this.now.format('X')))
    this.calculate()
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
      window.alert(`You must stake for more than ${this.rewardPeriod} days`)
      return
    }

    let startDateUnix = new Date(this.startDateTarget.value).getTime()
    let endDateUnix = new Date(this.endDateTarget.value).getTime()
    let url = `/stakingcalc/get-future-reward?startDate=${startDateUnix}&endDate=${endDateUnix}&startingBalance=${amount}`
    axios.get(url).then(function (response) {
      let result = response.data

      _this.daysTextTarget.textContent = parseInt(days)

      // number of periods
      const totalPercentage = result.reward
      const totalAmount = totalPercentage * amount * 1 / 100
      _this.percentageRoiTarget.textContent = totalPercentage.toFixed(2)
      _this.amountRoiTarget.textContent = totalAmount.toFixed(2)

      if (!result.simulation_table || result.simulation_table.length === 0) {
        hide(_this.tableTarget)
      } else {
        show(_this.tableTarget)
      }

      _this.tableBodyTarget.innerHTML = ''
      result.simulation_table.forEach(item => {
        const exRow = document.importNode(_this.rowTemplateTarget.content, true)
        const fields = exRow.querySelectorAll('td')

        let date = moment(startDateUnix).add(item.day, 'days')
        fields[0].innerText = date.format('YYYY-MM-DD')
        fields[1].innerText = item.height
        fields[2].innerText = item.ticket_price.toFixed(2)
        fields[3].innerText = item.returned_fund.toFixed(2)
        fields[4].innerText = item.reward.toFixed(2)
        fields[5].innerText = item.dcr_balance.toFixed(2)
        fields[6].innerText = (100 * (item.dcr_balance - amount) / amount).toFixed(2)
        fields[7].innerText = item.tickets_purchased
        _this.tableBodyTarget.appendChild(exRow)
      })
    })
  }
}
