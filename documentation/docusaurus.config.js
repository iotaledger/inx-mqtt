const path = require('path');

module.exports = {
    plugins: [
        [
            '@docusaurus/plugin-content-docs',
            {
                id: 'inx-mqtt',
                path: path.resolve(__dirname, 'docs'),
                routeBasePath: 'inx-mqtt',
                sidebarPath: path.resolve(__dirname, 'sidebars.js'),
                editUrl: 'https://github.com/gohornet/inx-mqtt/edit/develop/documentation/docs',
            }
        ],
    ],
    staticDirectories: [path.resolve(__dirname, 'static')],
};
