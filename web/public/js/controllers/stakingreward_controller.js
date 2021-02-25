import { Controller } from 'stimulus'
import axios from 'axios'
import moment from 'moment'

export default class extends Controller {
  static get targets () {
    return [
      'blockHeight', 'ticketPrice',
      'startDate', 'endDate',
      'priceDCR', 'dayText', 'amount', 'days', 'daysText',
      'amountRoi', 'percentageRoi',
      'tableBody', 'rowTemplate'
    ]
  }

  async connect () {
    this.ticketPrice = parseFloat(this.data.get('ticketPrice'))
    this.rewardPeriod = parseInt(this.data.get('rewardPeriod'))
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
      window.alert(`You must stake for more than ${this.rewardPeriod} days`)
      return
    }

    let startDateUnix = new Date(this.startDateTarget.value).getTime()
    let endDateUnix = new Date(this.endDateTarget.value).getTime()
    let url = `/staking-reward/get-future-reward?startDate=${startDateUnix}&endDate=${endDateUnix}&startingBalance=${amount}`
    axios.get(url).then(function (response) {
      let result = response.data

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
