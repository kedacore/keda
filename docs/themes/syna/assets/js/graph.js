import Chart from 'chart.js';
import $ from './helpers/jq-helpers';

const graphs = window.syna.api.getScope('graph');
Object.keys(graphs).forEach(key => {
  const config = graphs[key];
  window.syna.api.register('graphCharts', 'graphCharts-' + key, new Chart($(config.selector), {
    type: config.config.type || 'line',
    options: Object.assign({
      maintainAspectRatio: false,
    }, (config.config || {}).options),
    data: (config.config || {}).data,
  }));
});
