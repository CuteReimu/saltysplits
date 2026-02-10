const { createApp, ref, onMounted, nextTick } = Vue;
const { ElNotification } = ElementPlus;

const app = createApp({
    setup() {
        // --- 状态定义 ---
        const metaData = ref({}); // 对应原来的 data
        const summary = ref({});
        const disableShowBigSegment = ref(true);
        const showBigSegment = ref(false);
        const totalTimeType = ref("gameTime");
        const segmentOptions = ref([]);
        const selectedSegment = ref("");
        const segmentDetailData = ref({});

        // --- 内部数据 (不需要响应式) ---
        let totalRunDataRaw = null;
        let resetDataRaw = null;

        // --- Chart 实例 ---
        let totalRunChart = null;
        let resetChart = null;
        let segmentDataChart = null;

        // --- 辅助函数 ---

        // 将秒格式化为 hh:mm:ss
        const formatSecToHuman = (sec) => {
            if (sec == null || isNaN(sec)) return '';
            const totalSec = Math.floor(sec);
            const h = Math.floor(totalSec / 3600);
            const m = Math.floor((totalSec % 3600) / 60);
            const s = totalSec % 60;
            return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
        };

        // 将秒格式化为 mm:ss.xx 或 hh:mm:ss
        const formatSecToHuman2 = (sec) => {
            if (sec == null || isNaN(sec)) return '';
            let totalSec = sec;
            const h = Math.floor(totalSec / 3600);
            totalSec -= h * 3600;
            const m = Math.floor(totalSec / 60);
            totalSec -= m * 60;
            const s = Math.floor(totalSec);
            totalSec -= s;
            if (h === 0 && m < 10) {
                if (totalSec !== 0) {
                    const fractional = (totalSec * 100).toFixed(0).padStart(2, '0');
                    return `${m}:${String(s).padStart(2, '0')}.${fractional}`;
                }
                return `${m}:${String(s).padStart(2, '0')}`;
            }
            return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
        };

        // --- 图表创建逻辑 ---

        const createTotalRunData = () => {
            if (!totalRunDataRaw) return;
            const data = totalRunDataRaw[totalTimeType.value];
            const labels = data.map(item => String(item.id));
            const values = data.map(item => item.Time);

            if (totalRunChart) {
                totalRunChart.data.labels = labels;
                totalRunChart.data.datasets[0].data = values;
                totalRunChart.update();
                return;
            }

            const ctx = document.getElementById('totalTimeChart');
            if (!ctx) return;

            totalRunChart = new Chart(ctx, {
                type: 'line',
                data: {
                    labels,
                    datasets: [{
                        label: '总耗时',
                        data: values,
                        tension: 0.2,
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        x: {
                            type: 'linear',
                            title: { display: true, text: '尝试次数' },
                            ticks: { precision: 0 },
                            grid: { display: false },
                        },
                        y: {
                            type: 'linear',
                            title: { display: true, text: '总耗时' },
                            ticks: {
                                precision: 0,
                                callback: (value) => formatSecToHuman(value)
                            },
                            border: { display: false },
                        }
                    },
                    plugins: {
                        tooltip: {
                            callbacks: {
                                label: (context) => formatSecToHuman2(context.parsed.y)
                            }
                        },
                        legend: { display: false },
                        title: { display: true, text: '随时间推移的速通时长变化' }
                    }
                }
            });
        };

        const createResetData = () => {
            if (!resetDataRaw) return;
            const key = showBigSegment.value ? `${totalTimeType.value}Big` : totalTimeType.value;
            const data = resetDataRaw[key];
            if (!data) return;

            const labels = data.map(item => item.Segment);
            const values = data.map(item => item.Count);

            if (resetChart) {
                resetChart.data.labels = labels;
                resetChart.data.datasets[0].data = values;
                resetChart.update();
                return;
            }

            const ctx = document.getElementById('resetChart');
            if (!ctx) return;

            resetChart = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels,
                    datasets: [{
                        label: '重开次数',
                        data: values,
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { position: 'right' },
                        title: { display: true, text: '各分段重开次数' }
                    }
                }
            });
        };

        const createRunBreakdownData = (ctx, chartData) => {
            const labels = chartData.segments.map((seg, index) => index);
            const datasets = chartData.data.map(item => ({
                label: String(item.id),
                data: item.Details,
                tension: 0.2,
            }));

            new Chart(ctx, {
                type: 'line',
                data: { labels, datasets },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        x: {
                            type: 'linear',
                            title: { display: true, text: '用时' },
                            grid: { display: false },
                            ticks: {
                                precision: 0,
                                callback: (value) => formatSecToHuman(value)
                            },
                        },
                        y: {
                            type: 'category',
                            border: { display: false },
                            ticks: {
                                callback: (index) => chartData.segments[index]
                            },
                        }
                    },
                    plugins: {
                        tooltip: {
                            callbacks: {
                                title: () => "",
                                label: (context) => [
                                    `尝试: ${context.dataset.label}`,
                                    `分段: ${chartData.segments[context.formattedValue]}`,
                                    `用时: ${formatSecToHuman2(context.parsed.x)}`
                                ]
                            }
                        },
                        legend: {
                            position: 'top',
                            title: { display: true, text: '第 N 次尝试' }
                        },
                    }
                }
            });
        };

        const createSegmentData = (ctx, data) => {
            const labels = data.map(item => item.id);
            const values = data.map(item => item.Time);

            if (segmentDataChart) {
                segmentDataChart.data.labels = labels;
                segmentDataChart.data.datasets[0].data = values;
                segmentDataChart.update();
                return;
            }

            segmentDataChart = new Chart(ctx, {
                type: 'line',
                data: {
                    labels,
                    datasets: [{
                        label: '分段用时',
                        data: values,
                        tension: 0.2,
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        x: {
                            type: 'linear',
                            title: { display: true, text: '尝试次数' },
                            ticks: { precision: 0 },
                            grid: { display: false },
                        },
                        y: {
                            type: 'linear',
                            title: { display: true, text: '分段用时' },
                            border: { display: false },
                            ticks: {
                                precision: 2,
                                callback: (value) => formatSecToHuman2(value)
                            },
                        }
                    },
                    plugins: {
                        tooltip: {
                            callbacks: {
                                label: (context) => formatSecToHuman2(context.parsed.y)
                            }
                        },
                        legend: { display: false },
                    }
                }
            });
        };

        // --- 事件处理 ---

        const onTabChange = () => {
            createTotalRunData();
            createResetData();
        };

        const onSelectSegment = async (value) => {
            try {
                const response = await axios.get('/segment?index=' + value);
                const data = response.data;

                // 更新统计数据
                segmentDetailData.value = {
                    Min: data.Min,
                    Max: data.Max,
                    Average: data.Average,
                    Median: data.Median,
                    StandardDeviation: data.StandardDeviation,
                };

                // 更新图表
                // 确保 DOM 更新后再获取 canvas (虽然通常 select 改变时图表容器已存在)
                await nextTick();
                const ctx = document.getElementById('segmentData');
                if (ctx) {
                    createSegmentData(ctx, data.Details);
                }
            } catch (error) {
                console.error("Failed to fetch segment data:", error);
            }
        };

        // --- 生命周期钩子 ---

        onMounted(() => {
            // 显示开源通知
            ElNotification({
                dangerouslyUseHTMLString: true,
                message: '代码已在 Github 上开源：<a href="https://github.com/CuteReimu/saltysplits" rel="noopener noreferrer" target="_blank">https://github.com/CuteReimu/saltysplits</a>',
                position: 'bottom-right',
                duration: 0,
            });

            // 获取基础数据
            axios.get('/data')
                .then(res => metaData.value = res.data)
                .catch(err => console.error("Error fetching data:", err));

            axios.get('/summary')
                .then(res => summary.value = res.data)
                .catch(err => console.error("Error fetching summary:", err));

            // 获取总览数据并绘图
            axios.get('/totalData')
                .then(res => {
                    totalRunDataRaw = res.data;
                    createTotalRunData();
                })
                .catch(err => console.error("Error fetching totalData:", err));

            // 获取重置数据并绘图
            axios.get('/reset')
                .then(res => {
                    resetDataRaw = res.data;
                    disableShowBigSegment.value = res.data.disable;
                    createResetData();
                })
                .catch(err => console.error("Error fetching reset data:", err));

            // 获取细分数据并绘图
            axios.get('/breakdown')
                .then(async res => {
                    const data = res.data;
                    const rows = Array.isArray(data.segments) ? data.segments.length : 0;
                    const minHeight = 300;
                    const rowHeight = 24;
                    const h = Math.max(minHeight, rows * rowHeight);

                    const canvas = document.getElementById('runBreakdownData');
                    if (canvas) {
                        canvas.style.height = `${h}px`;
                        createRunBreakdownData(canvas, data);
                    }

                    // 初始化分段选择器
                    segmentOptions.value = data.segments.map((seg, index) => ({
                        label: seg,
                        value: index,
                    }));

                    if (segmentOptions.value.length > 0) {
                        selectedSegment.value = 0;
                        await onSelectSegment(0);
                    }
                })
                .catch(err => console.error("Error fetching breakdown:", err));
        });

        return {
            // State
            data: metaData, // 模板中使用了 data，这里为了兼容保持引用名为 data，但实际上是 metaData ref
            summary,
            disableShowBigSegment,
            showBigSegment,
            totalTimeType,
            segmentOptions,
            selectedSegment,
            segmentDetailData,

            // Methods
            onTabChange,
            onSelectSegment,
            createResetData // 模板中 v-model binding 可能会用到 watcher，或者 change handlers
        };
    }
});

app.use(ElementPlus);
app.mount('#app');

