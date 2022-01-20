const frequenciesPath = "./results/frequency/",
    postTopicsPath = "../../assets/so-data/posts-topic/",
    postsOperatorSearchPath = "../../assets/so-data/operators-search/",
    topics = [5, 19, 22],
    xValues = ["Introductory Questions", "iOS or Mobile", "Dependency Management"];

const RESULT_PATH = "./results",
    SIMILARITY_PATH = RESULT_PATH + "/similarity/",
    NUMBER_SAMPLES = 15;

const fs = require('fs'),
    fsPromisses = require('fs/promises'),
    path = require('path'),
    ChartJsImage = require('chartjs-to-image'),
    uniqolor = require('uniqolor'),
    csv = require('fast-csv'),
    d = require('distance-calc'),
    sm = require('sequencematcher');

// calculate simmilarity based on hamming algorithm
function calculateSimilarity(set1, set2) {
    let hamm = d.hamming(set1, set2);
    return 1 - (hamm / set1.length);
}

// get the rate of similarity
function getSimmilarityMatch(set1, set2) {
    let set1Map = new Map(set1);
    return [sm.sequenceMatcher(set1.map(([k, v]) => k), set2.map(([k, v]) => k)),
    set2.filter(([k, v]) => set1Map.has(k))]
}

// retunrs elements that share the same keys and are in the same position
function processSimilarity(set1, set2) {
    let set1Entries = [[...set1.keys()], [...set1.values()]];
    let set2Entries = [[...set2.keys()], [...set2.values()]];
    let finalMap = [];
    for (let i = 0; i < set1.size; i++) {
        if (set1Entries[0][i] == set2Entries[0][i] && set2Entries[1][i] > 0) {
            //console.log([set1Entries[0][i], set2Entries[1][i]]);
            finalMap.push([set1Entries[0][i], set2Entries[1][i]]);
        }
    }
    return new Map(finalMap);
}

// chart config
const config = {
    type: 'bar',
    options: {
        barValueSpacing: 10,
        legend: {
            display: true,
            position: "bottom",
            fontFamily: 'sans-serif',
            fontColor: "black"
        },
        title: {
            display: false,
        },
        scales: {
            xAxes: [{
                ticks: {
                    fontFamily: 'sans-serif',
                    fontStyle: 'bold',
                    fontColor: "black",
                    beginAtZero: true,
                }/* ,
                barThickness: 15,
                maxBarThickness: 15, */
            }],
            yAxes: [
                {
                    ticks: {
                        fontFamily: 'sans-serif',
                        fontStyle: 'bold',
                        beginAtZero: true,
                        fontColor: "black",
                        max: 20,
                        stepSize: 10,
                        callback: function (value) {
                            return (value / 100 * 100).toFixed(0) + '%'; // convert it to percentage
                        }
                    }
                }]
        }
    }
};

async function processFiles() {
    let postTopics = [];
    // load the Ids related to each topic
    for (let i = 0; i < topics.length; i++) {
        let rawData = await fsPromisses.readFile(path.resolve(postTopicsPath, `${topics[i]}.json`));
        postTopics.push(JSON.parse(rawData))
    }
    // maps to an array of maps to facilitate the search
    // innerVal == PostID
    postTopics = postTopics.map(postT => new Map(postT.map(innerVal => [innerVal, true])))
    // chart data
    var data1 = {
        labels: xValues,
        datasets: []
    }, data2 = {
        labels: xValues,
        datasets: []
    }, data3 = {
        labels: xValues,
        datasets: []
    };

    for (const val of ["RxJava", "RxJS", "RxSwift"]) {
        // load frequency data (GitHub)
        rawData = await fsPromisses.readFile(path.resolve(frequenciesPath, `frequencies_${val}.json`));
        let frequencyData = new Map(Object.entries(JSON.parse(rawData)));

        // reads the data from search of operators in the posts (Stack Overflow)
        const dirs = await fsPromisses.readdir(postsOperatorSearchPath);
        let file = "";
        dirs.some(d => {
            if (d.toUpperCase().includes(val.toUpperCase())) {
                file = d;
                return true;
            }
            return false
        });
        rawData = await fsPromisses.readFile(path.resolve(postsOperatorSearchPath, file));
        let operatorsResultData = JSON.parse(rawData);

        // makes an array of frequencies according to each topic
        let frequencySO = [new Map(), new Map(), new Map()];
        for (const [key, value] of Object.entries(operatorsResultData)) {
            for (const [op, total] of Object.entries(value)) {
                // calculates for each topic
                for (let top = 0; top < topics.length; top++) {
                    if (postTopics[top].has(parseInt(key))) {
                        if (!frequencySO[top].has(op)) {
                            frequencySO[top].set(op, 0)
                        }
                        frequencySO[top].set(op, frequencySO[top].get(op) + total)
                    }
                }
            }
        }
        // sort them descendingly
        for (let i = 0; i < frequencySO.length; i++) {
            frequencySO[i] = new Map([...frequencySO[i].entries()].sort((a, b) => b[1] - a[1]));
        }
        // datasetdata for similarity, similarity most used, and similarity least used
        let dataSetData1 = {
            label: val,
            fillColor: uniqolor(val).color,
            data: []
        }, dataSetData2 = {
            label: val,
            fillColor: uniqolor(val).color,
            data: []
        }, dataSetData3 = {
            label: val,
            fillColor: uniqolor(val).color,
            data: []
        }

        // write frequency to CSV
        for (let i = 0; i < frequencySO.length; i++) {
            writeCSV(`${SIMILARITY_PATH}/frequencies_${xValues[i]}_${val}`,
                ["Operator", "Frequency"], frequencySO[i])
            // takes advantage of the loop to write similarities
            let similarity =
                calculateSimilarity([...frequencyData.keys()], [...frequencySO[i].keys()]) * 100;
            dataSetData1.data.push(similarity);

            let similarityMap = processSimilarity(frequencyData, frequencySO[i]);
            writeCSV(`${SIMILARITY_PATH}/similarities_${xValues[i]}_${val}`,
                ["Operator", "Frequency"], similarityMap)

            // most used
            let simmilarMatch = getSimmilarityMatch(
                [...frequencyData.entries()].filter(([k, v]) => v > 0).slice(0, NUMBER_SAMPLES),
                [...frequencySO[i].entries()].filter(([k, v]) => v > 0).slice(0, NUMBER_SAMPLES)
            );
            dataSetData2.data.push(simmilarMatch[0] * 100);
            writeCSV(`${SIMILARITY_PATH}/similarities_mostUsed_${xValues[i]}_${val}`,
                ["Operator", "Frequency"], new Map(simmilarMatch[1]))
            // least used
            simmilarMatch = getSimmilarityMatch(
                [...frequencyData.entries()]
                    .filter(([k, v]) => v > 0).reverse().slice(0, NUMBER_SAMPLES).reverse(),
                [...frequencySO[i].entries()]
                    .filter(([k, v]) => v > 0).reverse().slice(0, NUMBER_SAMPLES).reverse());
            dataSetData3.data.push(simmilarMatch[0] * 100);
            writeCSV(`${SIMILARITY_PATH}/similarities_leastUsed_${xValues[i]}_${val}`,
                ["Operator", "Frequency"], new Map(simmilarMatch[1]))
        }
        // push data to chart data objs
        data1.datasets.push(dataSetData1);
        data2.datasets.push(dataSetData2);
        data3.datasets.push(dataSetData3);
    }
    // configuration for similarity chart
    config.data = data1;

    let myChart = new ChartJsImage();
    myChart.setConfig(config);

    //myChart.setHeight(400);
    myChart.setWidth(600);

    myChart.toFile(`${SIMILARITY_PATH}/similarity.png`);

    // configuration for similarity chart according to most used operators
    config.data = data2;
    config.options.scales.yAxes[0].ticks.max = 100;
    config.options.scales.yAxes[0].ticks.stepSize = 20;
    myChart = new ChartJsImage();
    myChart.setConfig(config);

    //myChart.setHeight(400);
    myChart.setWidth(600);

    myChart.toFile(`${SIMILARITY_PATH}/similarity_mostUsed.png`);
    
    // configuration for similarity chart according to least used operators
    config.data = data3;
    config.options.scales.yAxes[0].ticks.max = 60;
    config.options.scales.yAxes[0].ticks.stepSize = 20;
    myChart = new ChartJsImage();
    myChart.setConfig(config);

    //myChart.setHeight(400);
    myChart.setWidth(600);

    myChart.toFile(`${SIMILARITY_PATH}/similarity_leastUsed.png`);
}

function writeCSV(path, header, data) {
    const csvStream = csv.format({ writeBOM: true });
    csvStream
        .pipe(fs.createWriteStream(`${path}.csv`,
            { encoding: 'utf8' }))
        .on('finish', () => {
            console.log("Data written successfully to the CSV!");
        });
    // writes headers
    csvStream.write(header);
    for (const values of data) {
        csvStream.write(values);
    }
    csvStream.end();
}

// checks if results folder exists
if (!fs.existsSync(RESULT_PATH)) {
    fs.mkdirSync(RESULT_PATH)
}
if (fs.existsSync(SIMILARITY_PATH)) {
    fs.rmdirSync(SIMILARITY_PATH, { recursive: true })
}
fs.mkdirSync(SIMILARITY_PATH)
processFiles();