const fs = require('fs'),
    fsPromisses = require('fs/promises'),
    path = require('path'),
    ChartJsImage = require('chartjs-to-image');

const operatorsResult = require('./operators-result');

const RESULT_PATH = "./results",
    DISTRIBUTION_PATH = RESULT_PATH + "/utilization_perDistribution";

// chart config
const config = {
    type: 'bar',
    options: {
        legend: {
            display: true,
            position: "bottom",
            labels: {
                fontColor: "black",
                //fontStyle: "bold"
                fontFamily: "sans-serif"
            }
        },
        title: {
            display: false,
        }, plugins: {
            datalabels: {
                color: 'black',
                font: {
                    weight: 'bold',
                    family: 'sans-serif'
                },
                formatter: function (value, context) {
                    return value > 0 ? (value < 100 ? value.toFixed(1) + '%' : value + '%') : "";
                }
            }
        },
        scales: {
            xAxes: [{
                stacked: true,
                ticks: {
                    fontColor: 'black',
                    fontFamily: 'sans-serif',
                    fontStyle: 'bold'
                },
                //barThickness: 80,
                maxBarThickness: 80,
                /* categoryPercentage: 2.0,
                barPercentage: 2.0 */
            }],
            yAxes: [{
                stacked: true,
                ticks: {
                    fontColor: 'black',
                    fontFamily: 'sans-serif',
                    fontStyle: 'bold',
                    min: 0,
                    max: 100,
                    stepSize: 50,
                    callback: function (value) {
                        return (value / 100 * 100).toFixed(0) + '%'; // convert it to percentage
                    },
                }
            }]
        }
    }
};

async function processFiles() {
    let dist = ["RxJava", "RxJS", "RxSwift"]
    let beingUsed = [];
    let notBeingUsed = [];
    for (const val of dist) {
        // load result data
        rawData = await fsPromisses.readFile(path.resolve(operatorsResult.operatorsResultPath, operatorsResult[val].result));
        let operatorsResultData = JSON.parse(rawData);
        // creates a frequency map
        let frequencies = new Map();
        for (const value of Object.values(operatorsResultData)) {
            for ([op, total] of Object.entries(value)) {
                if (!frequencies.has(op)) {
                    frequencies.set(op, 0)
                }
                frequencies.set(op, frequencies.get(op) + total)
            }
        }

        beingUsed.push(([...frequencies.values()].filter(x => x > 0).length / frequencies.size) * 100)
        notBeingUsed.push(([...frequencies.values()].filter(x => x == 0).length / frequencies.size) * 100)
    }

    let data = {
        labels: dist,
        datasets: [{
            label: "Being used",
            backgroundColor: ["rgb(97,170,242)", "rgb(97,170,242)", "rgb(97,170,242)"],
            data: beingUsed
        }, {
            label: "Not being used",
            backgroundColor: ["rgb(242,194,145)", "rgb(242,194,145)", "rgb(242,194,145)"],
            data: notBeingUsed
        }]
    }

    config.data = data;
    const myChart = new ChartJsImage();
    myChart.setConfig(config);
    myChart.setHeight(200);
    myChart.setWidth(400);

    myChart.toFile(`${DISTRIBUTION_PATH}/utilization_perDistribution.png`);
}
// checks if results folder exists
if (!fs.existsSync(RESULT_PATH)) {
    fs.mkdirSync(RESULT_PATH)
}
if (fs.existsSync(DISTRIBUTION_PATH)) {
    fs.rmdirSync(DISTRIBUTION_PATH, { recursive: true })
}
fs.mkdirSync(DISTRIBUTION_PATH)
processFiles();