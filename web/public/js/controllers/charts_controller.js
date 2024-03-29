import { Controller } from 'stimulus'
import { map, assign, merge } from 'lodash-es'
import Zoom from '../helpers/zoom_helper'
import { darkEnabled } from '../services/theme_service'
import { animationFrame } from '../helpers/animation_helper'
import { getDefault } from '../helpers/module_helper'
import TurboQuery from '../helpers/turbolinks_helper'
import globalEventBus from '../services/event_bus_service'
import { isEqual } from '../helpers/chart_helper'
import dompurify from 'dompurify'
import humanize from '../helpers/humanize_helper'
import axios from 'axios'

var selectedChart
var chartResponse
let Dygraph // lazy loaded on connect

const aDay = 86400 * 1000 // in milliseconds
const aMonth = 30 // in days
const atomsToDCR = 1e-8
const windowScales = ['ticket-price', 'pow-difficulty', 'missed-votes']
const hybridScales = ['privacy-participation']
const lineScales = ['ticket-price', 'privacy-participation']
const modeScales = ['ticket-price']
const multiYAxisChart = ['ticket-price', 'coin-supply', 'privacy-participation']
// index 0 represents y1 and 1 represents y2 axes.
const yValueRanges = { 'ticket-price': [1] }
var chainworkUnits = ['exahash', 'zettahash', 'yottahash']
var hashrateUnits = ['Th/s', 'Ph/s', 'Eh/s']
var ticketPoolSizeTarget, premine, stakeValHeight, stakeShare
var baseSubsidy, subsidyInterval, subsidyExponent, windowSize, avgBlockTime
var rawCoinSupply, rawPoolValue
var yFormatter, legendEntry, legendMarker, legendElement

function usesWindowUnits (chart) {
  return windowScales.indexOf(chart) > -1
}

function usesHybridUnits (chart) {
  return hybridScales.indexOf(chart) > -1
}

function isScaleDisabled (chart) {
  return lineScales.indexOf(chart) > -1
}

function isModeEnabled (chart) {
  return modeScales.includes(chart)
}

function hasMultipleVisibility (chart) {
  return multiYAxisChart.indexOf(chart) > -1
}

function intComma (amount) {
  if (!amount) return ''
  return amount.toLocaleString(undefined, { maximumFractionDigits: 0 })
}

function axesToRestoreYRange (chartName, origYRange, newYRange) {
  let axesIndexes = yValueRanges[chartName]
  if (!Array.isArray(origYRange) || !Array.isArray(newYRange) ||
    origYRange.length !== newYRange.length || !axesIndexes) return

  var axes
  for (var i = 0; i < axesIndexes.length; i++) {
    let index = axesIndexes[i]
    if (newYRange.length <= index) continue
    if (!isEqual(origYRange[index], newYRange[index])) {
      if (!axes) axes = {}
      if (index === 0) {
        axes = Object.assign(axes, { y1: { valueRange: origYRange[index] } })
      } else if (index === 1) {
        axes = Object.assign(axes, { y2: { valueRange: origYRange[index] } })
      }
    }
  }
  return axes
}

function withBigUnits (v, units) {
  var i = v === 0 ? 0 : Math.floor(Math.log10(v) / 3)
  return (v / Math.pow(1000, i)).toFixed(3) + ' ' + units[i]
}

function blockReward (height) {
  if (height >= stakeValHeight) return baseSubsidy * Math.pow(subsidyExponent, Math.floor(height / subsidyInterval))
  if (height > 1) return baseSubsidy * (1 - stakeShare)
  if (height === 1) return premine
  return 0
}

function addLegendEntryFmt (div, series, fmt) {
  div.appendChild(legendEntry(`${series.dashHTML} ${series.labelHTML}: ${fmt(series.y)}`))
}

function addLegendEntry (div, series) {
  div.appendChild(legendEntry(`${series.dashHTML} ${series.labelHTML}: ${series.yHTML}`))
}

function defaultYFormatter (div, data) {
  addLegendEntry(div, data.series[0])
}

function customYFormatter (fmt) {
  return (div, data) => addLegendEntryFmt(div, data.series[0], fmt)
}

function legendFormatter (data) {
  if (data.x == null) return legendElement.classList.add('d-hide')
  legendElement.classList.remove('d-hide')
  var div = document.createElement('div')
  var xHTML = data.xHTML
  if (data.dygraph.getLabels()[0] === 'Date') {
    xHTML = humanize.date(data.x, false, selectedChart !== 'ticket-price')
  }
  div.appendChild(legendEntry(`${data.dygraph.getLabels()[0]}: ${xHTML}`))
  yFormatter(div, data, data.dygraph.getOption('legendIndex'))
  dompurify.sanitize(div, { IN_PLACE: true, FORBID_TAGS: ['svg', 'math'] })
  return div.innerHTML
}

function nightModeOptions (nightModeOn) {
  if (nightModeOn) {
    return {
      rangeSelectorAlpha: 0.3,
      gridLineColor: '#596D81',
      colors: ['#2DD8A3', '#2970FF', '#FFC84E']
    }
  }
  return {
    rangeSelectorAlpha: 0.4,
    gridLineColor: '#C4CBD2',
    colors: ['#2970FF', '#006600', '#FF0090']
  }
}

function zipWindowHvYZ (ys, zs, winSize, yMult, zMult, offset) {
  yMult = yMult || 1
  zMult = zMult || 1
  offset = offset || 0
  return ys.map((y, i) => {
    return [i * winSize + offset, y * yMult, zs[i] * zMult]
  })
}

function zipWindowHvY (ys, winSize, yMult, offset) {
  yMult = yMult || 1
  offset = offset || 0
  return ys.map((y, i) => {
    return [i * winSize + offset, y * yMult]
  })
}

function zipWindowTvYZ (times, ys, zs, yMult, zMult) {
  yMult = yMult || 1
  zMult = zMult || 1
  return times.map((t, i) => {
    return [new Date(t * 1000), ys[i] * yMult, zs[i] * zMult]
  })
}

function zipWindowTvY (times, ys, yMult) {
  yMult = yMult || 1
  return times.map((t, i) => {
    return [new Date(t * 1000), ys[i] * yMult]
  })
}

function zipTvY (times, ys, yMult) {
  yMult = yMult || 1
  return times.map((t, i) => {
    return [new Date(t * 1000), ys[i] * yMult]
  })
}

function zipIvY (ys, yMult, offset) {
  yMult = yMult || 1
  offset = offset || 1 // TODO: check for why offset is set to a default value of 1 when genesis block has a height of 0
  return ys.map((y, i) => {
    return [offset + i, y * yMult]
  })
}

function zipHvY (heights, ys, yMult, offset) {
  yMult = yMult || 1
  offset = offset || 1
  return ys.map((y, i) => {
    return [offset + heights[i], y * yMult]
  })
}

function zip2D (data, ys, yMult, offset) {
  yMult = yMult || 1
  if (data.axis === 'height') {
    if (data.bin === 'block') return zipIvY(ys, yMult)
    return zipHvY(data.h, ys, yMult, offset)
  }
  return zipTvY(data.t, ys, yMult)
}

function anonymitySetFunc (data) {
  let d
  let start = -1
  let end = 0
  if (data.axis === 'height') {
    if (data.bin === 'block') {
      d = data.anonymitySet.map((y, i) => {
        if (start === -1 && y > 0) {
          start = i
        }
        end = i
        return [i, y * atomsToDCR]
      })
    } else {
      d = data.anonymitySet.map((y, i) => {
        if (start === -1 && y > 0) {
          start = i
        }
        end = data.h[i]
        return [data.h[i], y * atomsToDCR]
      })
    }
  } else {
    d = data.t.map((t, i) => {
      if (start === -1 && data.anonymitySet[i] > 0) {
        start = t * 1000
      }
      end = t * 1000
      return [new Date(t * 1000), data.anonymitySet[i] * atomsToDCR]
    })
  }
  return { data: d, limits: [start, end] }
}

function ticketPriceFunc (data) {
  if (data.t) return zipWindowTvYZ(data.t, data.price, data.count, atomsToDCR)
  return zipWindowHvYZ(data.price, data.count, data.window, atomsToDCR)
}

function poolSizeFunc (data) {
  var out = []
  if (data.axis === 'height') {
    if (data.bin === 'block') out = zipIvY(data.count)
    else out = zipHvY(data.h, data.count)
  } else {
    out = zipTvY(data.t, data.count)
  }
  out.forEach(pt => pt.push(null))
  if (out.length) {
    out[0][2] = ticketPoolSizeTarget
    out[out.length - 1][2] = ticketPoolSizeTarget
  }
  return out
}

function percentStakedFunc (data) {
  rawCoinSupply = data.circulation.map(v => v * atomsToDCR)
  rawPoolValue = data.poolval.map(v => v * atomsToDCR)
  var ys = rawPoolValue.map((v, i) => [v / rawCoinSupply[i] * 100])
  if (data.axis === 'height') {
    if (data.bin === 'block') return zipIvY(ys)
    return zipHvY(data.h, ys)
  }
  return zipTvY(data.t, ys)
}

function powDiffFunc (data) {
  if (data.t) return zipWindowTvY(data.t, data.diff)
  return zipWindowHvY(data.diff, data.window)
}

function circulationFunc (chartData, showPredictedCurve) {
  var yMax = 0
  var h = -1
  let start = -1
  let end = 0
  var addDough = (newHeight) => {
    while (h < newHeight) {
      h++
      yMax += blockReward(h) * atomsToDCR
    }
  }
  var heights = chartData.h
  var times = chartData.t
  var supplies = chartData.supply
  var anonymitySet = chartData.anonymitySet
  var isHeightAxis = chartData.axis === 'height'
  var xFunc, hFunc
  if (chartData.bin === 'day') {
    xFunc = isHeightAxis ? i => heights[i] : i => new Date(times[i] * 1000)
    hFunc = i => heights[i]
  } else {
    xFunc = isHeightAxis ? i => i : i => new Date(times[i] * 1000)
    hFunc = i => i
  }

  var inflation = []
  var data = map(supplies, (n, i) => {
    if (isHeightAxis) {
      if (start === -1) {
        start = heights[i]
      }
      end = heights[i]
    } else {
      if (start === -1) {
        start = times[i] * 1000
      }
      end = times[i] * 1000
    }

    let height = hFunc(i)
    addDough(height)
    inflation.push(yMax)
    return [xFunc(i), supplies[i] * atomsToDCR, yMax, anonymitySet[i] * atomsToDCR]
  })

  var dailyBlocks = aDay / avgBlockTime
  var lastPt = data[data.length - 1]
  var x = lastPt[0]
  // Set yMax to the start at last actual supply for the prediction line.
  if (showPredictedCurve === true) {
    yMax = inflation[inflation.length - 1]
    if (!isHeightAxis) x = x.getTime()
    xFunc = isHeightAxis ? xx => xx : xx => { return new Date(xx) }
    var xIncrement = isHeightAxis ? dailyBlocks : aDay
    // Calculate inflation till 2045 => 25yrs from now => 300months
    var projection = 300 * aMonth
    for (var i = 1; i <= projection; i++) {
      addDough(h + dailyBlocks)
      x += xIncrement
      inflation.push(yMax)
      data.push([xFunc(x), null, yMax, null])
    }
  }
  return {
    data: data,
    inflation: inflation,
    limits: [start, end]
  }
}

function missedVotesFunc (data) {
  if (data.t) return zipWindowTvY(data.t, data.missed)
  return zipWindowHvY(data.missed, data.window, 1, data.offset * data.window)
}

function mapDygraphOptions (data, labelsVal, isDrawPoint, yLabel, labelsMG, labelsMG2) {
  return merge({
    'file': data,
    labels: labelsVal,
    drawPoints: isDrawPoint,
    ylabel: yLabel,
    labelsKMB: labelsMG2 && labelsMG ? false : labelsMG,
    labelsKMG2: labelsMG2 && labelsMG ? false : labelsMG2
  }, nightModeOptions(darkEnabled()))
}

export default class extends Controller {
  static get targets () {
    return [
      'chartWrapper',
      'labels',
      'chartsView',
      'chartSelect',
      'zoomSelector',
      'zoomOption',
      'scaleType',
      'axisOption',
      'binSelector',
      'scaleSelector',
      'ticketsPurchase',
      'ticketsPrice',
      'anonymitySet',
      'vSelectorItem',
      'vSelector',
      'binSize',
      'legendEntry',
      'legendMarker',
      'modeSelector',
      'modeOption',
      'rawDataURL',
      'supplySet',
      'predictedSet'
    ]
  }

  async connect () {
    this.query = new TurboQuery()
    ticketPoolSizeTarget = parseInt(this.data.get('tps'))
    premine = parseInt(this.data.get('premine'))
    stakeValHeight = parseInt(this.data.get('svh'))
    stakeShare = parseInt(this.data.get('pos')) / 10.0
    baseSubsidy = parseInt(this.data.get('bs'))
    subsidyInterval = parseInt(this.data.get('sri'))
    subsidyExponent = parseFloat(this.data.get('mulSubsidy')) / parseFloat(this.data.get('divSubsidy'))
    windowSize = parseInt(this.data.get('windowSize'))
    avgBlockTime = parseInt(this.data.get('blockTime')) * 1000
    legendElement = this.labelsTarget

    // Prepare the legend element generators.
    var lm = this.legendMarkerTarget
    lm.remove()
    lm.removeAttribute('data-target')
    legendMarker = () => {
      let node = document.createElement('div')
      node.appendChild(lm.cloneNode())
      return node.innerHTML
    }
    var le = this.legendEntryTarget
    le.remove()
    le.removeAttribute('data-target')
    legendEntry = s => {
      let node = le.cloneNode()
      node.innerHTML = s
      return node
    }

    this.defaultSettings = {
      chart: 'ticket-price',
      zoom: 'ikefq8bs-kavt2x8w',
      bin: 'window',
      axis: 'time',
      visibility: 'true-false-true',
      scale: 'linear',
      mode: 'smooth'
    }
    this.settings = TurboQuery.nullTemplate(['chart', 'zoom', 'scale', 'bin', 'axis', 'visibility', 'mode'])
    this.query.update(this.settings)
    this.settings.chart = this.settings.chart || 'ticket-price'
    this.zoomCallback = this._zoomCallback.bind(this)
    this.drawCallback = this._drawCallback.bind(this)
    this.limits = null
    this.lastZoom = null
    this.visibility = []
    if (this.settings.visibility) {
      this.settings.visibility.split(',', -1).forEach(s => {
        this.visibility.push(s === 'true')
      })
    }
    this.supplySetTarget.checked = this.settings.visibility ? this.settings.visibility[0] : true
    this.predictedSetTarget.checked = this.settings.visibility ? this.settings.visibility[1] : false
    this.anonymitySetTarget.checked = this.settings.visibility ? this.settings.visibility[2] : true
    this.lastSelectedZoom = ''
    Dygraph = await getDefault(
      import(/* webpackChunkName: "dygraphs" */ '../vendor/dygraphs.min.js')
    )
    this.drawInitialGraph()
    this.processNightMode = (params) => {
      this.chartsView.updateOptions(
        nightModeOptions(params.nightMode)
      )
    }
    globalEventBus.on('NIGHT_MODE', this.processNightMode)
  }

  disconnect () {
    globalEventBus.off('NIGHT_MODE', this.processNightMode)
    if (this.chartsView !== undefined) {
      this.chartsView.destroy()
    }
    selectedChart = null
  }

  updateQueryString () {
    const query = {}
    for (const k in this.settings) {
      if (!this.settings[k] || this.settings[k].toString() === this.defaultSettings[k].toString()) continue
      query[k] = this.settings[k]
    }
    this.query.replace(query)
  }

  drawInitialGraph () {
    var options = {
      axes: { y: { axisLabelWidth: 70 }, y2: { axisLabelWidth: 70 } },
      labels: ['Date', 'Ticket Price', 'Tickets Bought'],
      digitsAfterDecimal: 8,
      showRangeSelector: true,
      rangeSelectorPlotFillColor: '#8997A5',
      rangeSelectorAlpha: 0.4,
      rangeSelectorHeight: 40,
      drawPoints: true,
      pointSize: 0.25,
      legend: 'always',
      labelsSeparateLines: true,
      labelsDiv: legendElement,
      legendFormatter: legendFormatter,
      highlightCircleSize: 4,
      ylabel: 'Ticket Price',
      y2label: 'Tickets Bought',
      labelsUTC: true
    }

    this.chartsView = new Dygraph(
      this.chartsViewTarget,
      [[1, 1, 5], [2, 5, 11]],
      options
    )
    this.chartSelectTarget.value = this.settings.chart

    if (this.settings.axis) this.setAxis(this.settings.axis) // set first
    if (this.settings.scale === 'log') this.setScale(this.settings.scale)
    if (this.settings.zoom) this.setZoom(this.settings.zoom)
    this.setBin(this.settings.bin ? this.settings.bin : 'day')
    this.setMode(this.settings.mode ? this.settings.mode : 'smooth')

    var ogLegendGenerator = Dygraph.Plugins.Legend.generateLegendHTML
    Dygraph.Plugins.Legend.generateLegendHTML = (g, x, pts, w, row) => {
      g.updateOptions({ legendIndex: row }, true)
      return ogLegendGenerator(g, x, pts, w, row)
    }
    this.selectChart()
  }

  plotGraph (chartName, data) {
    var d = []
    var gOptions = {
      zoomCallback: null,
      drawCallback: null,
      logscale: this.settings.scale === 'log',
      valueRange: [null, null],
      visibility: null,
      y2label: null,
      stepPlot: this.settings.mode === 'stepped',
      axes: {},
      series: null,
      inflation: null
    }
    rawPoolValue = []
    rawCoinSupply = []
    yFormatter = defaultYFormatter
    var xlabel = data.t ? 'Date' : 'Block Height'

    switch (chartName) {
      case 'ticket-price': // price graph
        d = ticketPriceFunc(data)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Price', 'Tickets Bought'], true,
          'Price (DCR)', true, false))
        gOptions.y2label = 'Tickets Bought'
        gOptions.series = { 'Tickets Bought': { axis: 'y2' } }
        this.visibility = [this.ticketsPriceTarget.checked, this.ticketsPurchaseTarget.checked]
        gOptions.visibility = this.visibility
        gOptions.axes.y2 = {
          valueRange: [0, windowSize * 20 * 8],
          axisLabelFormatter: (y) => Math.round(y)
        }
        yFormatter = customYFormatter(y => y ? y.toFixed(8) + ' DCR' : '0.00000000 DCR')
        break

      case 'ticket-pool-size': // pool size graph
        d = poolSizeFunc(data)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Ticket Pool Size', 'Network Target'],
          false, 'Ticket Pool Size', true, false))
        gOptions.series = {
          'Network Target': {
            strokePattern: [5, 3],
            connectSeparatedPoints: true,
            strokeWidth: 2,
            color: '#888'
          }
        }
        yFormatter = customYFormatter(y => `${intComma(y)} tickets &nbsp;&nbsp; (network target ${intComma(ticketPoolSizeTarget)})`)
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'ticket-pool-value': // pool value graph
        d = zip2D(data, data.poolval, atomsToDCR)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Ticket Pool Value'], true,
          'Ticket Pool Value (DCR)', true, false))
        yFormatter = customYFormatter(y => intComma(y) + ' DCR')
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'stake-participation':
        d = percentStakedFunc(data)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Stake Participation'], true,
          'Stake Participation (%)', true, false))
        yFormatter = (div, data, i) => {
          addLegendEntryFmt(div, data.series[0], y => y.toFixed(4) + '%')
          div.appendChild(legendEntry(`${legendMarker()} Ticket Pool Value: ${intComma(rawPoolValue[i])} DCR`))
          div.appendChild(legendEntry(`${legendMarker()} Coin Supply: ${intComma(rawCoinSupply[i])} DCR`))
        }
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'block-size': // block size graph
        d = zip2D(data, data.size)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Block Size'], false, 'Block Size', true, false))
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'blockchain-size': // blockchain size graph
        d = zip2D(data, data.size)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Blockchain Size'], true,
          'Blockchain Size', false, true))
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'tx-count': // tx per block graph
        d = zip2D(data, data.count)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Number of Transactions'], false,
          '# of Transactions', false, false))
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'pow-difficulty': // difficulty graph
        d = powDiffFunc(data)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Difficulty'], true, 'Difficulty', true, false))
        break

      case 'coin-supply': // supply graph
        d = circulationFunc(data, this.predictedSetTarget.checked)
        this.customLimits = d.limits
        assign(gOptions, mapDygraphOptions(d.data, [xlabel, 'Coin Supply', 'Inflation Limit', 'Mix Rate'],
          true, 'Coin Supply (DCR)', true, false))
        gOptions.y2label = 'Inflation Limit'
        gOptions.y3label = 'Mix Rate'
        gOptions.series = { 'Inflation Limit': { axis: 'y2' }, 'Mix Rate': { axis: 'y3' } }
        this.visibility = [this.supplySetTarget.checked, this.predictedSetTarget.checked, this.anonymitySetTarget.checked]
        gOptions.visibility = this.visibility
        gOptions.series = {
          'Inflation Limit': {
            strokePattern: [5, 5],
            color: '#aaa',
            strokeWidth: 1.0
          },
          'Mix Rate': {
            color: '#2dd8a3'
          }
        }
        gOptions.inflation = d.inflation
        yFormatter = (div, data, i) => {
          if (data.series[0].y) {
            addLegendEntryFmt(div, data.series[0], y => intComma(y) + ' DCR')
          }
          var change = 0
          if (i < d.inflation.length) {
            const supply = data.series[0].y
            if (this.anonymitySetTarget.checked && data.series[0].y) {
              const mixed = data.series[2].y
              const mixedPercentage = ((mixed / supply) * 100).toFixed(2)
              div.appendChild(legendEntry(`${legendMarker()} Mixed: ${intComma(mixed)} DCR (${mixedPercentage}%)`))
            }
            let predicted = d.inflation[i]
            let unminted = predicted - (data.series[0].y || 0)
            change = ((unminted / predicted) * 100).toFixed(2)
            div.appendChild(legendEntry(`${legendMarker()} Unminted: ${intComma(unminted)} DCR (${change}%)`))
          }
        }
        break

      case 'fees': // block fee graph
        d = zip2D(data, data.fees, atomsToDCR)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Total Fee'], false, 'Total Fee (DCR)', true, false))
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'privacy-participation': // anonymity set graph
        d = anonymitySetFunc(data)
        this.customLimits = d.limits
        const label = 'Mix Rate'
        assign(gOptions, mapDygraphOptions(d.data, [xlabel, label], false, `${label} (DCR)`, true, false))

        yFormatter = (div, data, i) => {
          addLegendEntryFmt(div, data.series[0], y => y > 0 ? intComma(y) : '0' + ' DCR')
        }
        break

      case 'duration-btw-blocks': // Duration between blocks graph
        d = zip2D(data, data.duration, 1, 1)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Duration Between Blocks'], false,
          'Duration Between Blocks (seconds)', false, false))
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'chainwork': // Total chainwork over time
        d = zip2D(data, data.work)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Cumulative Chainwork (exahash)'],
          false, 'Cumulative Chainwork (exahash)', true, false))
        yFormatter = customYFormatter(y => withBigUnits(y, chainworkUnits))
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'hashrate': // Total chainwork over time
        d = zip2D(data, data.rate, 1e-3, data.offset)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Network Hashrate (petahash/s)'],
          false, 'Network Hashrate (petahash/s)', true, false))
        yFormatter = customYFormatter(y => withBigUnits(y * 1e3, hashrateUnits))
        this.defaultSettings.zoom = 'ikd7pc00-kauas5c0'
        this.defaultSettings.bin = 'day'
        break

      case 'missed-votes':
        d = missedVotesFunc(data)
        assign(gOptions, mapDygraphOptions(d, [xlabel, 'Missed Votes'], false,
          'Missed Votes per Window', true, false))
        this.defaultSettings.zoom = 'ikxndn6o-kavt2x8w'
        break
    }

    this.chartsView.plotter_.clear()
    this.chartsView.updateOptions(gOptions, false)
    if (yValueRanges[chartName]) this.supportedYRange = this.chartsView.yAxisRanges()
    this.validateZoom()
  }

  async selectChart () {
    var selection = this.settings.chart = this.chartSelectTarget.value
    this.customLimits = null
    this.chartWrapperTarget.classList.add('loading')
    if (isScaleDisabled(selection)) {
      this.scaleSelectorTarget.classList.add('d-hide')
      this.vSelectorTarget.classList.remove('d-hide')
    } else {
      this.scaleSelectorTarget.classList.remove('d-hide')
    }
    if (isModeEnabled(selection)) {
      this.modeSelectorTarget.classList.remove('d-hide')
    } else {
      this.modeSelectorTarget.classList.add('d-hide')
    }
    if (hasMultipleVisibility(selection)) {
      this.vSelectorTarget.classList.remove('d-hide')
      this.updateVSelector(selection)
    } else {
      this.vSelectorTarget.classList.add('d-hide')
    }
    if (selectedChart !== selection || this.settings.bin !== this.selectedBin() ||
      this.settings.axis !== this.selectedAxis()) {
      let url = '/api/charts/' + selection
      if (usesWindowUnits(selection) && !usesHybridUnits(selection)) {
        this.binSelectorTarget.classList.add('d-hide')
        this.settings.bin = 'window'
      } else {
        this.binSelectorTarget.classList.remove('d-hide')
        this.settings.bin = this.selectedBin()
        this.binSizeTargets.forEach(el => {
          if (el.dataset.option !== 'window') return
          if (usesHybridUnits(selection)) {
            el.classList.remove('d-hide')
          } else {
            el.classList.add('d-hide')
            if (this.settings.bin === 'window') {
              this.settings.bin = 'day'
              this.setActiveOptionBtn(this.settings.bin, this.binSizeTargets)
            }
          }
        })
      }
      url += `?bin=${this.settings.bin}`

      this.settings.axis = this.selectedAxis()
      if (!this.settings.axis) this.settings.axis = 'time' // Set the default.
      url += `&axis=${this.settings.axis}`
      this.setActiveOptionBtn(this.settings.axis, this.axisOptionTargets)
      chartResponse = await axios.get(url)
      selectedChart = selection
      this.plotGraph(selection, chartResponse.data)
    } else {
      this.chartWrapperTarget.classList.remove('loading')
    }
  }

  async validateZoom () {
    await animationFrame()
    this.chartWrapperTarget.classList.add('loading')
    await animationFrame()
    let oldLimits = this.limits || this.chartsView.xAxisExtremes()
    this.limits = this.chartsView.xAxisExtremes()
    var selected = this.selectedZoom()
    if (selected && !((selectedChart === 'privacy-participation' || selectedChart === 'coin-supply') && selected === 'all')) {
      if (selectedChart === 'coin-supply') {
        this.limits = oldLimits = [this.limits[0], this.customLimits[1]]
      }
      this.lastZoom = Zoom.validate(selected, this.limits,
        this.isTimeAxis() ? avgBlockTime : 1, this.isTimeAxis() ? 1 : avgBlockTime)
    } else {
      // if this is for the privacy-participation chart, then zoom to the beginning of the record
      if (selectedChart === 'privacy-participation') {
        this.limits = oldLimits = this.customLimits
        this.settings.zoom = Zoom.object(this.limits[0], this.limits[1])
      }
      if (selectedChart === 'coin-supply') {
        this.limits = oldLimits = [this.customLimits[0], this.limits[1]]
        this.settings.zoom = Zoom.object(this.limits[0], this.limits[1])
      }
      this.lastZoom = Zoom.project(this.settings.zoom, oldLimits, this.limits)
    }
    if (this.lastZoom) {
      this.chartsView.updateOptions({
        dateWindow: [this.lastZoom.start, this.lastZoom.end]
      })
    }
    if (selected !== this.lastSelectedZoom) {
      this.lastSelectedZoom = selected
      this._zoomCallback(this.lastZoom.start, this.lastZoom.end)
    }
    await animationFrame()
    this.chartWrapperTarget.classList.remove('loading')
    this.chartsView.updateOptions({
      zoomCallback: this.zoomCallback,
      drawCallback: this.drawCallback
    })
  }

  _zoomCallback (start, end) {
    this.lastZoom = Zoom.object(start, end)
    this.settings.zoom = Zoom.encode(this.lastZoom)
    this.updateQueryString()
    let ex = this.chartsView.xAxisExtremes()
    let option = Zoom.mapKey(this.settings.zoom, ex, this.isTimeAxis() ? 1 : avgBlockTime)
    this.setActiveOptionBtn(option, this.zoomOptionTargets)
    var axesData = axesToRestoreYRange(this.settings.chart,
      this.supportedYRange, this.chartsView.yAxisRanges())
    if (axesData) this.chartsView.updateOptions({ axes: axesData })
  }

  isTimeAxis () {
    return this.selectedAxis() === 'time'
  }

  _drawCallback (graph, first) {
    if (first) return
    var start, end
    [start, end] = this.chartsView.xAxisRange()
    if (start === end) return
    if (this.lastZoom.start === start) return // only handle slide event.
    this._zoomCallback(start, end)
  }

  setZoom (e) {
    var target = e.srcElement || e.target
    var option
    if (!target) {
      let ex = this.chartsView.xAxisExtremes()
      option = Zoom.mapKey(e, ex, this.isTimeAxis() ? 1 : avgBlockTime)
    } else {
      option = target.dataset.option
    }
    this.setActiveOptionBtn(option, this.zoomOptionTargets)
    if (!target) return // Exit if running for the first time
    this.validateZoom()
  }

  setBin (e) {
    var target = e.srcElement || e.target
    var option = target ? target.dataset.option : e
    if (!option) return
    this.setActiveOptionBtn(option, this.binSizeTargets)
    // hide vSelector
    this.updateVSelector()
    if (!target) return // Exit if running for the first time.
    selectedChart = null // Force fetch
    this.selectChart()
  }

  setScale (e) {
    var target = e.srcElement || e.target
    var option = target ? target.dataset.option : e
    if (!option) return
    this.setActiveOptionBtn(option, this.scaleTypeTargets)
    if (!target) return // Exit if running for the first time.
    if (this.chartsView) {
      this.chartsView.updateOptions({ logscale: option === 'log' })
    }
    this.settings.scale = option
    this.updateQueryString()
  }

  setMode (e) {
    var target = e.srcElement || e.target
    var option = target ? target.dataset.option : e
    if (!option) return
    this.setActiveOptionBtn(option, this.modeOptionTargets)
    if (!target) return // Exit if running for the first time.
    if (this.chartsView) {
      this.chartsView.updateOptions({ stepPlot: option === 'stepped' })
    }
    this.settings.mode = option
    this.updateQueryString()
  }

  setAxis (e) {
    var target = e.srcElement || e.target
    var option = target ? target.dataset.option : e
    if (!option) return
    this.setActiveOptionBtn(option, this.axisOptionTargets)
    if (!target) return // Exit if running for the first time.
    this.settings.axis = null
    this.selectChart()
  }

  updateVSelector (chart) {
    if (!chart) {
      chart = this.chartSelectTarget.value
    }
    const that = this
    let showWrapper = false
    this.vSelectorItemTargets.forEach(el => {
      let show = el.dataset.charts.indexOf(chart) > -1
      if (el.dataset.bin && el.dataset.bin.indexOf(that.selectedBin()) === -1) {
        show = false
      }
      if (show) {
        el.classList.remove('d-hide')
        showWrapper = true
      } else {
        el.classList.add('d-hide')
      }
    })
    if (showWrapper) {
      this.vSelectorTarget.classList.remove('d-hide')
    } else {
      this.vSelectorTarget.classList.add('d-hide')
    }
    this.setVisibilityFromSettings()
  }

  setVisibilityFromSettings () {
    switch (this.chartSelectTarget.value) {
      case 'ticket-price':
        if (this.visibility.length !== 2) {
          this.visibility = [true, this.ticketsPurchaseTarget.checked]
        }
        this.defaultSettings.visibility = 'true-false'
        this.ticketsPriceTarget.checked = this.visibility[0]
        this.ticketsPurchaseTarget.checked = this.visibility[1]
        break
      case 'coin-supply':
        if (this.visibility.length !== 3) {
          this.visibility = [this.supplySetTarget.checked, this.predictedSetTarget.checked, this.anonymitySetTarget.checked]
        }
        this.defaultSettings.bin = 'day'
        this.defaultSettings.zoom = 'ikd7pc00-khzi1hc0'
        this.supplySetTarget.checked = this.visibility[0]
        this.predictedSetTarget.checked = this.visibility[1]
        this.anonymitySetTarget.checked = this.visibility[2]
        break
      case 'privacy-participation':
        if (this.visibility.length !== 2) {
          this.defaultSettings.bin = 'day'
          this.defaultSettings.zoom = 'jzuht6o0-kauas5c0'
          this.settings.visibility = null
        }
        this.anonymitySetTarget.checked = this.visibility[1]
        break
      default:
        return
    }
    this.settings.visibility = this.visibility.join(',')
    this.updateQueryString()
  }

  setVisibility (e) {
    switch (this.chartSelectTarget.value) {
      case 'ticket-price':
        if (!this.ticketsPriceTarget.checked && !this.ticketsPurchaseTarget.checked) {
          e.currentTarget.checked = true
          return
        }
        this.visibility = [this.ticketsPriceTarget.checked, this.ticketsPurchaseTarget.checked]
        break
      case 'coin-supply':
        this.visibility = [this.supplySetTarget.checked, this.predictedSetTarget.checked, this.anonymitySetTarget.checked]
        break
      case 'privacy-participation':
        this.visibility = [true, this.anonymitySetTarget.checked]
        break
      default:
        return
    }
    if (e.target.dataset.target === this.predictedSetTarget.dataset.target) {
      this.plotGraph(selectedChart, chartResponse.data)
    } else {
      this.chartsView.updateOptions({ visibility: this.visibility })
    }
    this.settings.visibility = this.visibility.join(',')
    this.updateQueryString()
  }

  setActiveOptionBtn (opt, optTargets) {
    optTargets.forEach(li => {
      if (li.dataset.option === opt) {
        li.classList.add('active')
      } else {
        li.classList.remove('active')
      }
    })
  }

  selectedZoom () { return this.selectedOption(this.zoomOptionTargets) }
  selectedBin () { return this.selectedOption(this.binSizeTargets) }
  selectedScale () { return this.selectedOption(this.scaleTypeTargets) }
  selectedAxis () { return this.selectedOption(this.axisOptionTargets) }

  selectedOption (optTargets) {
    var key = false
    optTargets.forEach((el) => {
      if (el.classList.contains('active')) key = el.dataset.option
    })
    this.lastSelectedZoom = key
    return key
  }
}
