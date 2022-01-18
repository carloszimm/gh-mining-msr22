const fs = require('fs'),
    fsPromisses = require('fs/promises'),
    path = require('path'),
    ChartJsImage = require('chartjs-to-image');

const operatorsResult = require('./operators-result');

const RESULT_PATH = "./results",
    UTILIZATION_PATH = RESULT_PATH + "/utilization";

// chart config
const config = {
    type: 'pie',
    options: {
        legend: {
            display: true,
            position: "bottom",
            labels: {
                //fontColor: "black",
                //fontStyle: "bold"
                fontFamily: "Verdana"
            }
        },
        title: {
            display: false,
        },
        tooltips: {
            enabled: false
        }, plugins: {
            datalabels: {
                color: 'black',
                font: {
                    weight: 'bold',
                    family: 'Verdana'
                },
                formatter: function (value, context) {
                    let total = context.dataset.data[0] + context.dataset.data[1]
                    return ((parseInt(value) / total) * 100).toFixed(1) + '%';
                }
            }
        }
    }
};

async function processFiles() {
    // creates a frequency map
    let frequencies = new Map();
    for (const val of ["RxJava", "RxJS", "RxSwift"]) {
        // load result data
        rawData = await fsPromisses.readFile(path.resolve(operatorsResult.operatorsResultPath, operatorsResult[val].result));
        let operatorsResultData = JSON.parse(rawData);

        for (const value of Object.values(operatorsResultData)) {
            for ([op, total] of Object.entries(value)) {
                if (!frequencies.has(op)) {
                    frequencies.set(op, 0)
                }
                frequencies.set(op, frequencies.get(op) + total)
            }
        }
    }

    let data = {}

    let totalUsedOps = [...frequencies.values()].filter(x => x > 0).length
    let totalNotUsedOps = [...frequencies.values()].filter(x => x == 0).length
    let yValues = [totalUsedOps, totalNotUsedOps]
    let barColors = ["rgb(97,170,242)", "rgb(242,194,145)"];

    data.datasets = [{
        backgroundColor: barColors,
        data: yValues
    }];
    data.labels = ["Being used", "Not being used"];

    config.data = data;
    const myChart = new ChartJsImage();
    myChart.setConfig(config);

    myChart.toFile(`${UTILIZATION_PATH}/utilization.png`);
}
// checks if results folder exists
if (!fs.existsSync(RESULT_PATH)) {
    fs.mkdirSync(RESULT_PATH);
}
if (fs.existsSync(UTILIZATION_PATH)) {
    fs.rmdirSync(UTILIZATION_PATH, { recursive: true });
}
fs.mkdirSync(UTILIZATION_PATH);

processFiles();