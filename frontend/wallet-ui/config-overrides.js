const webpack = require('webpack');

module.exports = function override(config, env) {
  // Убеждаемся что resolve существует
  if (!config.resolve) {
    config.resolve = {};
  }
  
  // Настраиваем fallback для Node.js модулей
  // Важно: создаем новый объект, а не модифицируем существующий
  config.resolve.fallback = {
    ...config.resolve.fallback,
    "crypto": require.resolve("crypto-browserify"),
    "stream": require.resolve("stream-browserify"),
    "assert": require.resolve("assert"),
    "http": require.resolve("stream-http"),
    "https": require.resolve("https-browserify"),
    "os": require.resolve("os-browserify/browser"),
    "url": require.resolve("url"),
    "zlib": require.resolve("browserify-zlib"),
    "buffer": require.resolve("buffer"),
    "process": require.resolve("process/browser"),
    "vm": require.resolve("vm-browserify"),
    "util": require.resolve("util/"),
    "path": require.resolve("path-browserify"),
    "fs": false,
    "net": false,
    "tls": false,
  };
  
  // Добавляем плагины
  config.plugins = (config.plugins || []).concat([
    new webpack.ProvidePlugin({
      process: 'process/browser',
      Buffer: ['buffer', 'Buffer']
    })
  ]);
  
  // Отключаем предупреждения о source maps
  config.ignoreWarnings = [
    /Failed to parse source map/,
    /ENOENT: no such file or directory/
  ];
  
  // Отключаем source-map-loader для зависимостей чтобы избежать предупреждений
  if (config.module && config.module.rules) {
    config.module.rules = config.module.rules.filter(rule => {
      if (rule.use && Array.isArray(rule.use)) {
        return !rule.use.some(use => {
          if (typeof use === 'string') {
            return use === 'source-map-loader';
          }
          if (typeof use === 'object' && use.loader) {
            return use.loader === 'source-map-loader';
          }
          return false;
        });
      }
      return true;
    });
  }
  
  return config;
};

