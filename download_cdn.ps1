New-Item -ItemType Directory -Path static -Force
Invoke-WebRequest -Uri https://unpkg.com/vue@3.5.22/dist/vue.global.prod.js -OutFile static/vue.global.prod.js -v
Invoke-WebRequest -Uri https://unpkg.com/element-plus@2.11.3/dist/index.full.min.js -OutFile static/index.full.min.js -v
Invoke-WebRequest -Uri https://unpkg.com/element-plus@2.11.3/dist/index.css -OutFile static/index.css -v
Invoke-WebRequest -Uri https://unpkg.com/axios@1.12.2/dist/axios.min.js -OutFile static/axios.min.js -v
Invoke-WebRequest -Uri https://cdnjs.cloudflare.com/ajax/libs/Chart.js/4.5.0/chart.umd.min.js -OutFile static/chart.umd.min.js -v