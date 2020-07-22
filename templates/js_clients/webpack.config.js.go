package js_clients

const WEBPACK_CONFIG_JS = `
const path = require('path');
module.exports = {
    mode: 'development',
    entry: './main.js',
    devtool: 'source-map',
    output: {
        filename: 'main.js',
        path: path.resolve(__dirname, 'dist')
    },
    optimization: {
        // We no not want to minimize our code.
        minimize: false
    },
    devServer: {
        liveReload: true
    }
};
`
