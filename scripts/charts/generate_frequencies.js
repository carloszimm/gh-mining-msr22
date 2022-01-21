const fs = require('fs'),
  fsPromisses = require('fs/promises'),
  path = require('path'),
  ChartJsImage = require('chartjs-to-image'),
  uniqolor = require('uniqolor'),
  csv = require('fast-csv');

const operatorsResult = require('./operators-result');

const NUMBER_SAMPLES = 15,
  RESULT_PATH = "./results",
  FREQUENCY_PATH = RESULT_PATH + "/frequency";

// chart config
const config = {
  type: 'horizontalBar',
  options: {
    legend: { display: false },
    title: {
      display: false,
    },
    scales: {
      xAxes: [{
        ticks: {
          fontFamily: 'sans-serif',
          fontStyle: 'bold',
          beginAtZero: true,
          precision: 0,
          fontColor: "black"
        }
      }],
      yAxes: [
        {
          ticks: {
            fontFamily: 'sans-serif',
            fontStyle: 'bold',
            beginAtZero: true,
            fontColor: "black"
          }
        }]
    }
  }
};

async function processFiles() {
  for (const val of ["RxJava", "RxJS", "RxSwift"]) {
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
    let topFrequencies = [];
    // sort entries descendingly
    frequencies = new Map([...frequencies.entries()].sort((a, b) => b[1] - a[1]));
    // get first elements
    topFrequencies.push(new Map([...frequencies.entries()]
      .filter(([k, v]) => v > 0).slice(0, NUMBER_SAMPLES)));
    // get last elements
    topFrequencies.push(new Map([...frequencies.entries()]
      .filter(([k, v]) => v > 0).reverse().slice(0, NUMBER_SAMPLES).reverse()));
    
    for (let i = 0; i < topFrequencies.length; i++) {
      let data = {
        labels: [],
        datasets: []
      }
      let yValues = [], barColors = []
      for (const [key, value] of topFrequencies[i]) {
        data.labels.push(key);
        yValues.push(value);
        barColors.push(uniqolor(key).color);
      };
      data.datasets.push({
        backgroundColor: barColors,
        data: yValues
      })
      config.data = data;
      const myChart = new ChartJsImage();
      /* let max = yValues[0];
      if (max == 1){
        config.options.scales.xAxes[0].ticks.max = (max + 0.2);
      } else{
        config.options.scales.xAxes[0].ticks.max = undefined;
      } */
      myChart.setConfig(config);

      myChart.setHeight(250);
      //myChart.setWidth(600);

      myChart.toFile(`${FREQUENCY_PATH}/frequecy_${val}_top${i === 0 ? "MostUsed" : "LeastUsed"}.png`);
    }
    // writes extracted data to JSON
    await fsPromisses.writeFile(`${FREQUENCY_PATH}/frequencies_${val}.json`,
      JSON.stringify(Object.fromEntries(frequencies)));
    // writes extracted data to CSV
    const csvStream = csv.format({ writeBOM: true });
    csvStream
      .pipe(fs.createWriteStream(`${FREQUENCY_PATH}/frequencies_${val}.csv`, { encoding: 'utf8' }))
      .on('finish', () => {
        console.log("Data written successfully to the CSV and JSON files!");
      });
    // writes headers
    csvStream.write(["Operator", "Frequency"]);
    for (const values of frequencies) {
      csvStream.write(values);
    }
    csvStream.end();
  }
}
// checks if results folder exists
if (!fs.existsSync(RESULT_PATH)) {
  fs.mkdirSync(RESULT_PATH)
}
if (fs.existsSync(FREQUENCY_PATH)) {
  fs.rmdirSync(FREQUENCY_PATH, { recursive: true })
}
fs.mkdirSync(FREQUENCY_PATH)
processFiles();