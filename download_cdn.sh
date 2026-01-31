#!/bin/sh
mkdir -p static
curl https://unpkg.com/vue@3.5.22/dist/vue.global.prod.js -o static/vue.global.prod.js
curl https://unpkg.com/element-plus@2.11.3/dist/index.full.min.js -o static/index.full.min.js
curl https://unpkg.com/element-plus@2.11.3/dist/index.css -o static/index.css
curl https://unpkg.com/axios@1.12.2/dist/axios.min.js -o static/axios.min.js
curl https://cdnjs.cloudflare.com/ajax/libs/Chart.js/4.5.0/chart.umd.min.js -o static/chart.umd.min.js