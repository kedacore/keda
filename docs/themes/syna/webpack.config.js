const path = require('path');

module.exports = {
  entry: {
    head: './assets/js/head.js',
    main: './assets/js/index.js',
    collapse: './assets/js/collapse.js',
    contact: './assets/js/contact.js',
    graph: './assets/js/graph.js',
    hero: './assets/js/hero.js',
    portfolio: './assets/js/portfolio.js',
    pricing: './assets/js/pricing.js',
    react: './assets/js/react.js',
    search: './assets/js/search.js',
    stripe: './assets/js/stripe.js',
  },
  output: {
    path: path.resolve('./assets/scripts/'),
    filename: 'syna-[id].js'
  },
  resolve: {
    extensions: ['.js', '.jsx'],
    modules: [path.join(process.cwd(), 'src'), 'node_modules'],
  },
  mode: process.env === 'production' ? 'production' : 'development',
  module: {
    rules: [{
      test: /\.jsx?$/,
      exclude: /node_modules/,
      use: {
        loader: 'babel-loader',
        options: {
          presets: ['env', 'react'],
        },
      },
    }],
  },
  plugins: [],
};
