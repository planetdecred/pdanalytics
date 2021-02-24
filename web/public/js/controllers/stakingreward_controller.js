import { Controller } from 'stimulus'
import axios from 'axios'
import moment from 'moment'

export default class extends Controller {
  static get targets () {
    return [
      'blockHeight', 'ticketPrice',
      'startDate', 'endDate',
      'priceDCR', 'dayText', 'amount', 'amountText', 'days', 'daysText',
      'amountRoi', 'percentageRoi', 'tickets',
      'tableBody', 'rowTemplate'
    ]
  }

  async connect () {
    this.ticketPrice = parseFloat(this.data.get('ticketPrice'))
    this.startDateTarget.value = moment().subtract(1, 'year').format('YYYY-MM-DD')
    this.endDateTarget.value = moment().format('YYYY-MM-DD')
    this.amountTarget.value = 1000
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
      _this.daysTextTarget.textContent = parseInt(days)

      // number of periods
      const totalPercentage = result.reward
      const totalAmount = totalPercentage * amount * 1 / 100
      _this.percentageRoiTarget.textContent = totalPercentage.toFixed(2)
      _this.amountRoiTarget.textContent = totalAmount.toFixed(2)

      _this.tableBodyTarget.innerHTML = ''
      result.simulation_table.forEach(item => {
        const exRow = document.importNode(_this.rowTemplateTarget.content, true)
        const fields = exRow.querySelectorAll('td')

        fields[0].innerText = item.height
        fields[1].innerText = item.returned_fund.toFixed(2)
        fields[2].innerText = item.reward.toFixed(2)
        fields[3].innerText = item.dcr_balance.toFixed(2)
        fields[4].innerText = item.ticket_price.toFixed(4)
        fields[5].innerText = item.tickets_purchased
        _this.tableBodyTarget.appendChild(exRow)
      })
    })
  }
}
