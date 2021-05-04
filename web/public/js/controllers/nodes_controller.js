import { Controller } from 'stimulus'
import axios from 'axios'
import {
  hide,
  hideLoading,
  insertOrUpdateQueryParam, legendFormatter, options,
  selectedOption,
  setActiveOptionBtn,
  show,
  showLoading,
  updateQueryParam,
  hideAll,
  trimUrl,
  csv,
  notifyFailure,
  updateZoomSelector
} from '../utils'

import TurboQuery from '../helpers/turbolinks_helper'
import { animationFrame } from '../helpers/animation_helper'
import Zoom from '../helpers/zoom_helper'
import humanize from '../helpers/humanize_helper'

const Dygraph = require('../vendor/dygraphs.min.js')

const dataTypeNodes = 'nodes'
const dataTypeVersion = 'version'
const dataTypeLocation = 'location'

export default class extends Controller {
  timestamp
  nextTimestamp
  previousTimestamp
  height
  currentPage
  pageSize
  userAgentsPage
  query
  selectedViewOption

  static get targets () {
    return [
      'viewOption', 'chartDataTypeSelector', 'chartDataType',
      'numPageWrapper', 'pageSize', 'messageView', 'chartWrapper', 'chartsView', 'labels',
      'btnWrapper', 'nextPageButton', 'previousPageButton', 'tableTitle', 'tableWrapper', 'tableHeader', 'tableBody',
      'snapshotRowTemplate', 'userAgentRowTemplate', 'countriesRowTemplate', 'totalPageCount', 'currentPage', 'loadingData',
      'dataTypeSelector', 'dataType', 'chartSourceWrapper', 'chartSource', 'chartsViewWrapper', 'chartSourceList',
      'allChartSource', 'graphIntervalWrapper', 'interval', 'zoomSelector', 'zoomOption'
    ]
  }

  initialize () {
    // Turbolinks' cache control causes the initialize method to be fired multiple time
    // because the preview is loaded first before the actual content is gotten from the
    // server. If this is a preview, do nothing
    if (this.data.get('cached') === '1') return
    this.data.set('cached', '1')

    this.currentPage = parseInt(this.data.get('page')) || 1
    this.pageSize = parseInt(this.data.get('pageSize')) || 20
    this.selectedViewOption = this.data.get('viewOption')
    this.dataType = this.data.get('dataType') || dataTypeNodes
    setActiveOptionBtn(this.dataType, this.dataTypeTargets)

    this.query = new TurboQuery()
    this.settings = TurboQuery.nullTemplate([
      'zoom', 'bin', 'axis', 'dataType', 'page', 'view-option'
    ])
    this.query.update(this.settings)

    if (this.settings.zoom) {
      setActiveOptionBtn(this.settings.zoom, this.zoomOptionTargets)
    }
    if (this.settings.bin) {
      setActiveOptionBtn(this.settings.bin, this.intervalTargets)
    }
    this.zoomCallback = this._zoomCallback.bind(this)
    this.drawCallback = this._drawCallback.bind(this)

    this.userAgentsPage = 1
    this.countriesPage = 1
    this.updateView()
  }

  async updateView () {
    if (this.selectedViewOption === 'table') {
      this.setTable()
    } else {
      await this.setChart()
    }
  }

  async setTable () {
    this.selectedViewOption = 'table'
    setActiveOptionBtn(this.selectedViewOption, this.viewOptionTargets)
    hide(this.chartWrapperTarget)
    hide(this.zoomSelectorTarget)
    hide(this.messageViewTarget)
    hide(this.graphIntervalWrapperTarget)
    show(this.tableWrapperTarget)
    show(this.numPageWrapperTarget)
    insertOrUpdateQueryParam('view-option', this.selectedViewOption, 'chart')
    this.reloadTable()
    trimUrl(['view-option', 'data-type', 'page', 'page-size'])
  }

  async setChart () {
    this.selectedViewOption = 'chart'
    hide(this.tableWrapperTarget)
    hide(this.messageViewTarget)
    await this.populateChartSources()
    setActiveOptionBtn(this.selectedViewOption, this.viewOptionTargets)
    setActiveOptionBtn(this.dataType, this.chartDataTypeTargets)
    hide(this.numPageWrapperTarget)
    show(this.chartWrapperTarget)
    show(this.graphIntervalWrapperTarget)
    updateQueryParam('view-option', this.selectedViewOption, 'chart')
    this.reloadChat()
    trimUrl(['view-option', 'data-type', 'zoom', 'bin'])
    // reset this table properties as the url params will be reset
    this.currentPage = 1
    this.pageSizeTarget.value = this.pageSize = 20
  }

  allChartSourceCheckChanged (event) {
    const checked = event.currentTarget.checked
    this.chartSourceTargets.forEach(el => {
      el.checked = checked
    })
    this.reloadChat()
  }

  chartSourceCheckChanged (event) {
    let count = 0
    this.chartSourceTargets.forEach(el => {
      if (el.checked) {
        count++
      }
    })
    if (count > 5) {
      event.currentTarget.checked = false
      notifyFailure('You cannot compare more than 5 sources')
      return
    }
    this.reloadChat()
  }

  updateChartUI () {
    switch (this.dataType) {
      case dataTypeNodes:
        hide(this.chartSourceWrapperTarget)
        this.chartsViewWrapperTarget.classList.remove('col-md-21')
        this.chartsViewWrapperTarget.classList.remove('col-md-10')
        this.chartsViewWrapperTarget.classList.add('col-md-24')
        return
      case dataTypeLocation:
        this.chartsViewWrapperTarget.classList.remove('col-md-20')
        this.chartsViewWrapperTarget.classList.add('col-md-21')
        this.chartsViewWrapperTarget.classList.remove('col-md-24')
        this.chartSourceWrapperTarget.classList.add('col-md-3')
        this.chartSourceWrapperTarget.classList.remove('col-md-4')
        break
      case dataTypeVersion:
        this.chartsViewWrapperTarget.classList.add('col-md-20')
        this.chartsViewWrapperTarget.classList.remove('col-md-21')
        this.chartsViewWrapperTarget.classList.remove('col-md-24')
        this.chartSourceWrapperTarget.classList.add('col-md-4')
        this.chartSourceWrapperTarget.classList.remove('col-md-3')
        break
    }
    show(this.chartSourceWrapperTarget)
  }

  async populateChartSources () {
    let url = `/api/snapshot/${this.dataType === dataTypeVersion ? 'node-versions' : 'node-countries'}`
    showLoading(this.loadingDataTarget, [this.tableWrapperTarget])
    const _this = this
    let response = await axios.get(url)
    let result = response.data
    hideLoading(_this.loadingDataTarget)
    if (result.error) {
      let messageHTML = `<div class="alert alert-primary"><strong>${result.error}</strong></div>`
      _this.messageViewTarget.innerHTML = messageHTML
      show(_this.messageViewTarget)
      hide(_this.chartWrapperTarget)
      return
    }
    show(_this.chartWrapperTarget)
    let html = ''
    _this.allChartSourceTarget.checked = true
    result.forEach((item, i) => {
      var checked = i <= 4
      if (item === '') {
        checked = false
        item = 'Unknown'
      }
      html += `<div class="form-check">
                    <input name="chartSource" data-target="nodes.chartSource" data-action="click->nodes#chartSourceCheckChanged"
                    class="form-check-input" type="checkbox" id="inlineCheckbox-${item}" value="${item}" ${checked ? 'checked' : ''}>
                    <label class="form-check-label" for="inlineCheckbox-${item}">${item}</label>
                </div>`
    })
    _this.chartSourceListTarget.innerHTML = html
  }

  async setDataType (e) {
    this.dataType = e.currentTarget.getAttribute('data-option')
    if (this.dataType === selectedOption(this.dataTypeTargets)) {
      return
    }
    this.currentPage = 1
    insertOrUpdateQueryParam('page', this.currentPage, 1)
    setActiveOptionBtn(this.dataType, this.dataTypeTargets)
    insertOrUpdateQueryParam('data-type', this.dataType, 'nodes')
    await this.populateChartSources()
    this.updateView()
  }

  loadNextPage () {
    this.currentPage += 1
    insertOrUpdateQueryParam('page', this.currentPage, 1)
    this.reloadTable()
  }

  loadPreviousPage () {
    this.currentPage -= 1
    if (this.currentPage < 1) {
      this.currentPage = 1
    }
    insertOrUpdateQueryParam('page', this.currentPage, 1)
    this.reloadTable()
  }

  reloadTable () {
    let url
    let displayFn
    switch (this.dataType) {
      case dataTypeVersion:
        url = '/api/snapshots/user-agents'
        displayFn = this.displayUserAgents
        break
      case dataTypeLocation:
        url = '/api/snapshots/countries'
        displayFn = this.displayCountries
        break
      case dataTypeNodes:
      default:
        url = '/api/snapshots'
        displayFn = this.displaySnapshotTable
        break
    }
    const _this = this
    showLoading(this.loadingDataTarget, [_this.tableWrapperTarget])
    url += `?page=${this.currentPage}&page-size=${this.pageSize}`
    axios.get(url).then(function (response) {
      let result = response.data
      hideLoading(_this.loadingDataTarget, [_this.tableWrapperTarget])
      if (result.error) {
        let messageHTML = `<div class="alert alert-primary"><strong>${result.error}</strong></div>`
        _this.messageViewTarget.innerHTML = messageHTML
        show(_this.messageViewTarget)
        hide(_this.tableBodyTarget)
        hide(_this.btnWrapperTarget)
        return
      }
      hide(_this.messageViewTarget)
      show(_this.tableBodyTarget)
      show(_this.btnWrapperTarget)
      _this.totalPageCountTarget.textContent = result.totalPages
      _this.currentPageTarget.textContent = _this.currentPage

      if (_this.currentPage <= 1) {
        hide(_this.previousPageButtonTarget)
      } else {
        show(_this.previousPageButtonTarget)
      }

      if (_this.currentPage >= result.totalPages) {
        hide(_this.nextPageButtonTarget)
      } else {
        show(_this.nextPageButtonTarget)
      }
      displayFn = displayFn.bind(_this)
      displayFn(result)
    }).catch(function (e) {
      hideLoading(_this.loadingDataTarget)
      console.log(e) // todo: handle error
    })
  }

  displayUserAgents (result) {
    this.tableTitleTarget.innerHTML = 'User Agents'
    this.showHeader(dataTypeVersion)
    this.tableBodyTarget.innerHTML = ''

    const _this = this
    result.userAgents.forEach(item => {
      const exRow = document.importNode(_this.userAgentRowTemplateTarget.content, true)
      const fields = exRow.querySelectorAll('td')

      fields[0].innerText = humanize.date(item.timestamp * 1000)
      fields[1].innerText = item.user_agent
      fields[2].innerText = item.nodes

      _this.tableBodyTarget.appendChild(exRow)
    })
  }

  displayCountries (result) {
    this.tableTitleTarget.innerHTML = 'Countries'
    this.showHeader(dataTypeLocation)
    this.tableBodyTarget.innerHTML = ''

    const _this = this
    result.countries.forEach(item => {
      const exRow = document.importNode(_this.countriesRowTemplateTarget.content, true)
      const fields = exRow.querySelectorAll('td')

      fields[0].innerText = humanize.date(item.timestamp * 1000)
      fields[1].innerText = item.country || 'Unknown'
      fields[2].innerText = item.nodes

      _this.tableBodyTarget.appendChild(exRow)
    })
  }

  displaySnapshotTable (result) {
    this.tableTitleTarget.innerHTML = 'Network Snapshots'
    this.showHeader(dataTypeNodes)
    this.tableBodyTarget.innerHTML = ''

    const _this = this
    result.data.forEach(item => {
      const exRow = document.importNode(_this.snapshotRowTemplateTarget.content, true)
      const fields = exRow.querySelectorAll('td')

      fields[0].innerText = humanize.date(item.timestamp * 1000)
      fields[1].innerText = item.node_count
      fields[2].innerText = item.reachable_node_count

      _this.tableBodyTarget.appendChild(exRow)
    })
  }

  showHeader (dataType) {
    hideAll(this.tableHeaderTargets)
    this.tableHeaderTargets.forEach(el => {
      if (el.getAttribute('data-for') === dataType) {
        show(el)
      }
    })
  }

  changePageSize (e) {
    this.pageSize = parseInt(e.currentTarget.value)
    insertOrUpdateQueryParam('page-size', this.pageSize, 20)
    this.reloadTable()
  }

  // chart
  selectedZoom () { return selectedOption(this.zoomOptionTargets) }

  setZoom (e) {
    var target = e.srcElement || e.target
    var option
    if (!target) {
      let ex = this.chartsView.xAxisExtremes()
      option = Zoom.mapKey(e, ex, 1)
    } else {
      option = target.dataset.option
    }
    setActiveOptionBtn(option, this.zoomOptionTargets)
    if (!target) return // Exit if running for the first time
    insertOrUpdateQueryParam('zoom', option, 'all')
    this.validateZoom()
  }

  async validateZoom () {
    await animationFrame()
    await animationFrame()
    let oldLimits = this.limits || this.chartsView.xAxisExtremes()
    this.limits = this.chartsView.xAxisExtremes()
    var selected = this.selectedZoom()
    if (selected) {
      this.lastZoom = Zoom.validate(selected, this.limits, 1, 1)
    } else {
      this.lastZoom = Zoom.project(this.settings.zoom, oldLimits, this.limits)
    }
    if (this.lastZoom) {
      this.chartsView.updateOptions({
        dateWindow: [this.lastZoom.start, this.lastZoom.end]
      })
    }
    if (selected !== this.settings.zoom) {
      this._zoomCallback(this.lastZoom.start, this.lastZoom.end)
    }
    await animationFrame()
    this.chartsView.updateOptions({
      zoomCallback: this.zoomCallback,
      drawCallback: this.drawCallback
    })
  }

  _zoomCallback (start, end) {
    this.lastZoom = Zoom.object(start, end)
    this.settings.zoom = Zoom.encode(this.lastZoom)
    let ex = this.chartsView.xAxisExtremes()
    let option = Zoom.mapKey(this.settings.zoom, ex, 1)
    setActiveOptionBtn(option, this.zoomOptionTargets)
  }

  _drawCallback (graph, first) {
    if (first) return
    var start, end
    [start, end] = this.chartsView.xAxisRange()
    if (start === end) return
    if (this.lastZoom.start === start) return // only handle slide event.
    this._zoomCallback(start, end)
  }

  selectedInterval () { return selectedOption(this.intervalTargets) }

  setInterval (e) {
    const option = e.currentTarget.dataset.option
    setActiveOptionBtn(option, this.intervalTargets)
    insertOrUpdateQueryParam('bin', option, 'day')
    this.reloadChat()
  }

  async reloadChat () {
    let url
    let drawChartFn

    this.selectedSources = []
    this.chartSourceTargets.forEach(el => {
      if (el.checked) {
        this.selectedSources.push(el.value)
      }
    })
    let q = `bin=${this.selectedInterval()}`
    if (this.selectedSources.length > 0) {
      q += `&extras=${this.selectedSources.join('|')}`
    }
    switch (this.dataType) {
      case dataTypeVersion:
        url = `/api/charts/snapshot/node-versions?${q}`
        drawChartFn = this.drawUserAgentsChart
        break
      case dataTypeLocation:
        url = `/api/charts/snapshot/locations?${q}`
        drawChartFn = this.drawCountriesChart
        break
      case dataTypeNodes:
      default:
        url = `/api/charts/snapshot/nodes?${q}`
        drawChartFn = this.drawSnapshotChart
        break
    }

    this.drawInitialGraph()
    showLoading(this.loadingDataTarget)
    const response = await axios.get(url)
    const result = response.data
    if (result.error) {
      this.messageViewTarget.innerHTML = `<div class="alert alert-primary"><strong>${result.error}</strong></div>`
      show(this.messageViewTarget)
      hide(this.chartWrapperTarget)
      hideLoading(this.loadingDataTarget)
      return
    }
    if (!result.x || result.x.length === 0) {
      this.messageViewTarget.innerHTML = `<div class="alert alert-primary"><strong>No record for the selected chart</strong></div>`
      show(this.messageViewTarget)
      hide(this.chartWrapperTarget)
      hideLoading(this.loadingDataTarget)
      return
    }
    show(this.chartWrapperTarget)
    hide(this.messageViewTarget)
    drawChartFn = drawChartFn.bind(this)
    this.updateChartUI()
    drawChartFn(result)
  }

  drawSnapshotChart (result) {
    this.chartsView = new Dygraph(
      this.chartsViewTarget,
      csv(result, 2),
      {
        legend: 'always',
        includeZero: true,
        // dateWindow: [minDate, maxDate],
        legendFormatter: legendFormatter,
        digitsAfterDecimal: 8,
        labelsDiv: this.labelsTarget,
        ylabel: 'Node Count',
        y2label: 'Reachable Nodes',
        xlabel: 'Date',
        labels: ['Date', 'Node Count', 'Reachable Nodes'],
        labelsUTC: true,
        labelsKMB: true,
        maxNumberWidth: 10,
        showRangeSelector: true,
        axes: {
          x: {
            drawGrid: false
          },
          y: {
            axisLabelWidth: 90
          }
        }
      }
    )
    hideLoading(this.loadingDataTarget)
    this.validateZoom()
    let minDate, maxDate
    result.x.forEach(unixTime => {
      let date = new Date(unixTime * 1000)
      if (minDate === undefined || date < minDate) {
        minDate = date
      }

      if (maxDate === undefined || date > maxDate) {
        maxDate = date
      }
    })
    if (updateZoomSelector(this.zoomOptionTargets, minDate, maxDate)) {
      show(this.zoomSelectorTarget)
    } else {
      hide(this.zoomSelectorTarget)
    }
  }

  drawUserAgentsChart (result) {
    let options = {
      legend: 'always',
      includeZero: true,
      legendFormatter: legendFormatter,
      labelsDiv: this.labelsTarget,
      ylabel: 'Node Count',
      xlabel: 'Date (UTC)',
      labels: ['Date (UTC)', ...this.selectedSources],
      labelsUTC: true,
      labelsKMB: true,
      connectSeparatedPoints: true,
      showRangeSelector: true,
      axes: {
        x: {
          drawGrid: false
        }
      }
    }
    this.chartsView = new Dygraph(
      this.chartsViewTarget,
      csv(result, this.selectedSources.length),
      options
    )
    hideLoading(this.loadingDataTarget)
    this.validateZoom()
    let minDate, maxDate
    result.x.forEach(unixTime => {
      let date = new Date(unixTime * 1000)
      if (minDate === undefined || date < minDate) {
        minDate = date
      }

      if (maxDate === undefined || date > maxDate) {
        maxDate = date
      }
    })
    if (updateZoomSelector(this.zoomOptionTargets, minDate, maxDate)) {
      show(this.zoomSelectorTarget)
    } else {
      hide(this.zoomSelectorTarget)
    }
  }

  drawCountriesChart (result) {
    let options = {
      legend: 'always',
      includeZero: true,
      legendFormatter: legendFormatter,
      labelsDiv: this.labelsTarget,
      ylabel: 'Node Count',
      xlabel: 'Date (UTC)',
      labels: ['Date (UTC)', ...this.selectedSources],
      labelsUTC: true,
      labelsKMB: true,
      connectSeparatedPoints: true,
      showRangeSelector: true,
      axes: {
        x: {
          drawGrid: false
        }
      }
    }
    const data = csv(result, this.selectedSources.length)
    this.chartsView = new Dygraph(
      this.chartsViewTarget,
      data,
      options
    )
    hideLoading(this.loadingDataTarget)
    this.validateZoom()
    let minDate, maxDate
    result.x.forEach(unixTime => {
      let date = new Date(unixTime * 1000)
      if (minDate === undefined || date < minDate) {
        minDate = date
      }

      if (maxDate === undefined || date > maxDate) {
        maxDate = date
      }
    })
    if (updateZoomSelector(this.zoomOptionTargets, minDate, maxDate)) {
      show(this.zoomSelectorTarget)
    } else {
      hide(this.zoomSelectorTarget)
    }
  }

  drawInitialGraph () {
    var extra = {
      legendFormatter: legendFormatter,
      labelsDiv: this.labelsTarget,
      ylabel: 'Node Count',
      xlabel: 'Date',
      labels: ['Date', 'Node Count'],
      labelsUTC: true,
      labelsKMB: true,
      axes: {
        x: {
          drawGrid: false
        }
      }
    }

    this.chartsView = new Dygraph(
      this.chartsViewTarget,
      [[0, 0]],
      { ...options, ...extra }
    )
  }
}
