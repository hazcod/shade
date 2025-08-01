const path = require('path');
const CopyPlugin = require('copy-webpack-plugin');

module.exports = (env, argv) => {
  const isProduction = argv.mode === 'production';
  
  return {
    mode: argv.mode || 'development',
    entry: {
      background: './src/background/index.ts',
      content: './src/content/index.ts',
      popup: './src/popup/index.ts',
    },
    output: {
      path: path.resolve(__dirname, 'dist'),
      filename: '[name].js',
    },
    resolve: {
      extensions: ['.ts', '.js'],
        fallback: {
          crypto: false,
          buffer: false,
          stream: false,
          path: false,
          vm: false,
          fs: false,
        },
    },
    module: {
      rules: [
        {
          test: /\.ts$/,
          use: 'ts-loader',
          exclude: /node_modules/,
        },
      ],
    },
    devtool: isProduction ? 'source-map' : 'source-map',
    optimization: {
      minimize: isProduction,
    },
    plugins: [
      new CopyPlugin({
        patterns: [
          { from: 'manifest.json', to: '.' },
          { from: 'public', to: '.' },
          { from: 'src/popup/popup.html', to: '../popup.html' },
        ],
      }),
    ],
  };
};