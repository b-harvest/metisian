import React, { useState, useEffect, useRef } from 'react';
import UIkit from 'uikit';
import escape from 'lodash/escape';
import logo from "../../assets/images/logo.ico";
import { Col } from "antd"
import { useSeqStatus } from "../../context/statusContext";


const h = 24
const w = 9
const textMax = 115
const textW = 120
let gridH = h
let gridW = w
let gridTextMax = textMax
let gridTextW = textW
let scale = 1
let textColor = "#3f3f3f"

let signColorAlpha = 0.2

function fix_dpi(id) {
    let canvas = document.getElementById(id),
        dpi = window.devicePixelRatio;
    gridH = h * dpi.valueOf()
    gridW = w * dpi.valueOf()
    gridTextMax = textMax * dpi.valueOf()
    gridTextW = textW * dpi.valueOf()
    let style = {
        height() {
            return +getComputedStyle(canvas).getPropertyValue('height').slice(0,-2);
        },
        width() {
            return +getComputedStyle(canvas).getPropertyValue('width').slice(0,-2);
        }
    }
    canvas.setAttribute('width', style.width() * dpi);
    canvas.setAttribute('height', style.height() * dpi);
    scale = dpi.valueOf()
}

function legend() {
    const l = document.getElementById("legend")
    l.height = scale * h * 1.2
    const ctx = l.getContext('2d')

    let offset = textW
    let grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, 'rgb(123,255,66)');
    grad.addColorStop(0.3, 'rgb(240,255,128)');
    grad.addColorStop(0.8, 'rgb(169,250,149)');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    ctx.font = `${scale * 14}px sans-serif`
    ctx.fillStyle = 'grey'
    offset += gridW + gridW/2
    ctx.fillText("proposer",offset, gridH/1.2)

    offset += 65 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, 'rgba(0,0,0,0.2)');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    ctx.fillStyle = 'grey'
    offset += gridW + gridW/2
    ctx.fillText("signed",offset, gridH/1.2)

    offset += 50 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#85c0f9');
    grad.addColorStop(0.7, '#85c0f9');
    grad.addColorStop(1, '#0b2641');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("miss/precommit",offset, gridH/1.2)

    offset += 110 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#381a34');
    grad.addColorStop(0.2, '#d06ec7');
    grad.addColorStop(1, '#d06ec7');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("miss/prevote", offset, gridH/1.2)

    offset += 90 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#8e4b26');
    grad.addColorStop(0.4, 'darkorange');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    ctx.beginPath();
    ctx.moveTo(offset + 1, gridH-2-gridH/2);
    ctx.lineTo(offset + 4 + gridW / 4, gridH-1-gridH/2);
    ctx.closePath();
    ctx.strokeStyle = 'white'
    ctx.stroke();
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("missed", offset, gridH/1.2)

    offset += 59 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, 'rgba(127,127,127,0.3)');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("no data", offset, gridH/1.2)
}

function drawSeries(multiStates) {
    const canvas = document.getElementById("canvas")
    if (canvas === null) {
        return
    }
    canvas.height = ((12*gridH*multiStates.Status.length)/10) + 30
    fix_dpi("canvas")
    if (canvas.getContext) {
        const ctx = canvas.getContext('2d')
        ctx.font = `${scale * 16}px sans-serif`
        ctx.fillStyle = textColor

        let crossThrough = false
        for (let j = 0; j < multiStates.Status.length; j++) {

            //ctx.fillStyle = 'white'
            ctx.fillStyle = textColor
            ctx.fillText(multiStates.Status[j].name, 5, (j*gridH)+(gridH*2)-6, gridTextMax)

            for (let i = 0; i < multiStates.Status[j].blocks.length; i++) {
                crossThrough = false
                const grad = ctx.createLinearGradient((i*gridW)+gridTextW, (gridH*j), (i * gridW) + gridW +gridTextW, (gridH*j))
                switch (multiStates.Status[j].blocks[i]) {
                    case 4: // proposed
                        grad.addColorStop(0, 'rgb(123,255,66)');
                        grad.addColorStop(0.3, 'rgb(240,255,128)');
                        grad.addColorStop(0.8, 'rgb(169,250,149)');
                        break
                    case 3: // signed
                        if (j % 2 === 0) {
                            grad.addColorStop(0, `rgba(0,0,0,${signColorAlpha})`);
                            grad.addColorStop(0.9, `rgba(0,0,0,${signColorAlpha})`);
                        } else {
                            grad.addColorStop(0, `rgba(0,0,0,${signColorAlpha-0.3})`);
                            grad.addColorStop(0.9, `rgba(0,0,0,${signColorAlpha-0.3})`);
                        }
                        grad.addColorStop(1, 'rgb(186,186,186)');
                        break
                    case 2: // precommit not included
                        grad.addColorStop(0, '#85c0f9');
                        grad.addColorStop(0.8, '#85c0f9');
                        grad.addColorStop(1, '#0b2641');
                        break
                    case 1: // prevote not included
                        grad.addColorStop(0, '#381a34');
                        grad.addColorStop(0.2, '#d06ec7');
                        grad.addColorStop(1, '#d06ec7');
                        break
                    case 0: // missed
                        grad.addColorStop(0, '#fa9d9d');
                        crossThrough = true
                        break
                    default:
                        grad.addColorStop(0, 'rgba(127,127,127,0.3)');
                }
                ctx.clearRect((i*gridW)+gridTextW, gridH+(gridH*j), gridW, gridH)
                ctx.fillStyle = grad
                ctx.fillRect((i*gridW)+gridTextW, gridH+(gridH*j), gridW, gridH)

                // line between rows
                if (i > 0) {
                    ctx.beginPath();
                    ctx.moveTo((i * gridW) - gridW + gridTextW, 2 * gridH + (gridH * j) - 0.5)
                    ctx.lineTo((i * gridW) + gridTextW, 2 * gridH + (gridH * j) - 0.5);
                    ctx.closePath();
                    ctx.strokeStyle = 'rgb(51,51,51)'
                    ctx.strokeWidth = '5px;'
                    ctx.stroke();
                }

                // visual differentiation for missed blocks
                if (crossThrough) {
                    ctx.beginPath();
                    ctx.moveTo((i * gridW) + gridTextW + 1 + gridW / 4, (gridH*j) + (gridH * 2) - gridH / 2);
                    ctx.lineTo((i * gridW) + gridTextW + gridW - (gridW / 4) - 1, (gridH*j) + (gridH * 2) - gridH / 2);
                    ctx.closePath();
                    ctx.strokeStyle = 'white'
                    ctx.stroke();
                }
            }
        }
    }
}


const SignStatus = () => {
    const { statusData, setStatusData } = useSeqStatus(); 
  const [logs, setLogs] = useState([]);
  const logRef = useRef(null);
  const HOST = import.meta.env.VITE_HOST? import.meta.env.VITE_HOST: "localhost:8888/"
  const PROTOCOL = "http://"

  useEffect(() => {
    connect();
  }, [])

  useEffect(() => {
    UIkit.icon();
    loadState();
  }, []);

  const loadState = async () => {
    try {
      const enableLogsResponse = await fetch(PROTOCOL + HOST + "logsenabled", {
        method: 'GET',
        mode: 'cors',
        cache: 'no-cache',
        credentials: 'same-origin',
        redirect: 'error',
        referrerPolicy: 'no-referrer'
      });
      const enableLogs = await enableLogsResponse.json();
      if (enableLogs.enabled === false) {
        document.getElementById("logContainer").hidden = true;
      }

      const stateResponse = await fetch(PROTOCOL + HOST + "state", {
        method: 'GET',
        mode: 'cors',
        cache: 'no-cache',
        credentials: 'same-origin',
        redirect: 'error',
        referrerPolicy: 'no-referrer'
      });
      const initialState = await stateResponse.json();
      updateTable(initialState);
      drawSeries(initialState);

      const logResponse = await fetch(PROTOCOL + HOST + "logs", {
        method: 'GET',
        mode: 'cors',
        cache: 'no-cache',
        credentials: 'same-origin',
        redirect: 'error',
        referrerPolicy: 'no-referrer'
      });
      const logData = await logResponse.json();
      const formattedLogs = logData.map(log => `${new Date(log.ts * 1000).toLocaleTimeString()} - ${log.msg}`);
      setLogs(formattedLogs);
    } catch (error) {
      console.error(error);
    }
  };

  
  const updateTable = (status) => {
    setStatusData(status)
    const table = document.getElementById("statusTable");
    if (!table) return;
  
    // Clear existing rows
    table.innerHTML = '';

    status.Status.forEach((item, i) => {
        let alerts = "&nbsp;";
        if (item.active_alerts > 0 || item.last_error !== "") {
            let alerts = "&nbsp;";
            let modalId = encodeURIComponent(item.name);
            let lastError = encodeURIComponent(item.last_error);

            if (item.active_alerts > 0 || item.last_error !== "") {
            if (item.last_error !== "") {
                alerts = `
                <a href="#modal-center-${modalId}" uk-toggle>
                    <span uk-icon='warning' uk-tooltip="${item.active_alerts} active issues" style='color: darkorange'></span>
                </a>
                <div id="modal-center-${modalId}" class="uk-flex-top" uk-modal>
                    <div class="uk-modal-dialog uk-modal-body uk-margin-auto-vertical uk-background-secondary">
                    <button class="uk-modal-close-default" type="button" uk-close></button>
                    <pre class="uk-background-secondary" style="color: white">${lastError}</pre>
                    </div>
                </div>
                `;
            } else {
                alerts = `<span uk-icon='warning' uk-tooltip="${item.active_alerts} active issues" style='color: darkorange'></span>`;
            }
            }
            UIkit.update();
        }
        
      const validBlocks = item.blocks.filter(block => block !== -1);
      
      const missedBlocks = validBlocks.filter(block => block >= 3).length;
      const missedPercentage = (missedBlocks / validBlocks.length) * 100;
      const formattedPercentage = missedPercentage.toFixed(2);
  
      const row = table.insertRow();
      row.insertCell(0).innerHTML = `<div>${alerts}</div>`;
      row.insertCell(1).innerHTML = item.name === "not connected"
      ? `<div class="uk-text-warning">${escape(item.name)}</div>`
      : `<div class='uk-text-truncate'>${escape(item.name.substring(0, 24))}</div>`;
      row.insertCell(2).innerHTML = `<div>${escape(item.address)}</div>`;
      row.insertCell(3).innerHTML = `<div>${formattedPercentage}</div>`;
    });
  };
  

  const addLogMsg = (msg) => {
    setLogs(prevLogs => {
      const newLogs = [msg, ...prevLogs];
      if (newLogs.length > 256) newLogs.pop();
      if (document.visibilityState !== "hidden") {
        logRef.current.innerText = newLogs.join("\n");
      }
      return newLogs;
    });
  };

  const connect = () => {
    const wsProto = location.protocol === "https:" ? "wss://" : "ws://";
    const socket = new WebSocket(wsProto + HOST + 'ws');

    socket.addEventListener('message', (event) => {
      const msg = JSON.parse(event.data);
      if (msg.msgType === "log") {
        addLogMsg(`${new Date(msg.ts * 1000).toLocaleTimeString()} - ${msg.msg}`);
      } else if (msg.msgType === "update" && document.visibilityState !== "hidden") {
        updateTable(msg);
        drawSeries(msg);
      }
    });

    socket.onclose = (e) => {
      console.log('Socket is closed, retrying /ws ...', e.reason);
      addLogMsg('Socket is closed, retrying /ws ...' + e.reason);
      setTimeout(connect, 3000);
    };
  };


  return (
    <div className="uk-container uk-width-expand uk-height-viewport">
      <div className="uk-width-expand uk-overflow-auto uk-background-default" id="canvasDiv">
        <div style={{ width: '4835px' }} className="uk-padding-remove-horizontal">
          <canvas id="canvas" height="20" width="4735"></canvas>
        </div>
        <div id="legendContainer" className="uk-nav-center uk-background-default uk-padding-remove">
          <canvas id="legend" height="32" width="700"></canvas>
        </div>
      </div>

      <div className="uk-padding-small uk-text-small uk-background-default uk-overflow-auto" id="tableDiv">
        <table className="uk-table uk-table-small uk-table-justify uk-padding-remove">
          <thead>
            <tr>
              <th></th>
              <th className="uk-text-center">Name</th>
              <th className="uk-text-center">Address</th>

              <th className="uk-text-center">Tendermint Uptime(%)</th>
            </tr>
          </thead>
          <tbody id="statusTable"></tbody>
        </table>
      </div>

      <div className="uk-padding-large uk-padding-remove-top" id="logContainer">
      <pre
        ref={logRef}
        id="logs"
        className="uk-panel-scrollable uk-resize-vertical"
        style={{ 
            height: '300px',
            outline: '2px solid #000000',
            background: '#f2f2f2'
        }}
        ></pre>

      </div>

      <div className="uk-text-small uk-text-center">
        <ul/>
        <Col>
            <span>Developed and Maintained by B-Harvest. </span>
            <a className="uk-link-muted" href="https://bharvest.io">
                <img src={logo} alt="[ B-Harvest ]" style={{ height: '24px' }} />
            </a>
        </Col>
        <li/>
        <span>Forked by Tenderduty which developed by blockpane</span>
      </div>
    </div>
  );
};

export default SignStatus;
