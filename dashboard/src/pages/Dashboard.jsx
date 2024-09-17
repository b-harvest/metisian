import {React, useContext, useEffect, useState } from 'react';

import {
  Card,
  Col,
  Row,
  Spin,
} from "antd";

import 'react-horizontal-scrolling-menu/dist/styles.css';
import { Link } from "react-router-dom";
import '../assets/styles/app.css'

import { Banner } from '../components/Banner'

import SignStatus from "../components/SignStatus";
import { useSeqStatus } from '../context/statusContext';

const {Meta} = Card

const MAINNET_SEQUENCER_RESOURCE = "https://raw.githubusercontent.com/MetisProtocol/metis-sequencer-resources/main/sequencers/1088";
const SEPOLIA_SEQUENCER_RESOURCE_BASE = "https://raw.githubusercontent.com/MetisProtocol/metis-sequencer-resources/main/sequencers/59902";
const ALL_ENDPOINT = "/all.json";

export const SEQUENCER_EXPLORER = "https://sequencer.metis.io/#/sequencers/"


export const truncateText = (text, maxLength) => {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength) + '...';
};


function Dashboard() {
  const { statusData } = useSeqStatus();
  const [seqResourceData, setSeqResourceData] = useState([]);
  const [epochStat, setEpochStat] = useState([]);
  const [sortedStatus, setSortedStatus] = useState([]);

  // useEffect(() => console.log(statusData), [statusData])

  useEffect(() => {
    loadSeqResource()
    // console.log(seqResourceData); // debug only
  }, [])

  const loadSeqResource = async () => {
    const seqResource = await fetch(SEPOLIA_SEQUENCER_RESOURCE_BASE + ALL_ENDPOINT, {
      method: 'GET',
      mode: 'cors',
      cache: 'no-cache',
      credentials: 'same-origin',
      redirect: 'error',
      referrerPolicy: 'no-referrer'
    });
    const seqResData = await seqResource.json();
    setSeqResourceData(seqResData);
  }


  useEffect(() => {
    if (!statusData?.Status || !seqResourceData) {
      return;
    }
  
    const newEpochStat = [];
  
    statusData.Status.forEach((item) => {
      seqResourceData.forEach((resourceItem) => {
        if (item.address === resourceItem.seq_addr.toLowerCase()) {
          newEpochStat.push({
            address: item.address,
            avatar: resourceItem.avatar.replace("{BASEDIR}", SEPOLIA_SEQUENCER_RESOURCE_BASE),
            id: item.latest_selected_epoch,
            name: item.name,
            desc: resourceItem.desc
          });
        }
      });
    });
  
    setEpochStat(newEpochStat);
    setSortedStatus([...newEpochStat].sort((a, b) => a.id - b.id));

  }, [statusData, seqResourceData]);
  
  
  
  return (
    <>
      <div className="space30"></div>
      <div className="banner h-410"></div>
        <div className="layout-content">
        <Row gutter={[32, 32]} justify="center">
          <Col>
          <Link to={sortedStatus.length > 2 ? SEQUENCER_EXPLORER + sortedStatus[sortedStatus.length - 3].address:""}>
            <Card
            title={sortedStatus.length > 2 ?"Epoch: " + sortedStatus[sortedStatus.length - 3].id:""}
              loading={sortedStatus.length > 2 ?null:"true"}
              key={"previous-seq"}
              hoverable
              style={{ width: 270, height: 450 }}
              bordered={false}
              cover={
                <img
                  alt={sortedStatus.length > 2 ?sortedStatus[sortedStatus.length - 3].address:""}
                  src={sortedStatus.length > 2 ?sortedStatus[sortedStatus.length - 3].avatar:""}
                  style={{ height: 270, objectFit: "cover" }}
                /> 
              }
              className="criclebox"
            >
              <Meta 
              loading={sortedStatus.length > 2 ?null:"true"}
              title={sortedStatus.length > 2 ?sortedStatus[sortedStatus.length - 3].name:""} 
              description={sortedStatus.length > 2 ?truncateText(sortedStatus[sortedStatus.length - 3].desc,80):""}/>
            </Card>
            </Link>
          </Col>
          <Col>
            <Link to={sortedStatus.length > 1 ? SEQUENCER_EXPLORER + sortedStatus[sortedStatus.length - 2].address:""}>
            <Card
            title={sortedStatus.length > 1 ?"Epoch: " + sortedStatus[sortedStatus.length - 2].id:""}
              loading={sortedStatus.length > 1 ?null:"true"}
              key={"current-seq"}
              hoverable 
              style={{ width: 300, height: 500}} 
              bordered={false} 
              cover={
                <img 
                  alt={sortedStatus.length > 1 ?sortedStatus[sortedStatus.length - 2].address:""} 
                  src={sortedStatus.length > 1 ?sortedStatus[sortedStatus.length - 2].avatar:""} 
                  style={{ height: 300, objectFit: "cover" }}
                />
              }
              className="criclebox"
            >
              <Meta 
              loading={sortedStatus.length > 1 ?null:"true"}
              title={sortedStatus.length > 1 ?sortedStatus[sortedStatus.length - 2].name:""} 
              description={sortedStatus.length > 1 ?truncateText(sortedStatus[sortedStatus.length - 2].desc,100):""}/>
            </Card>
            </Link>
          </Col>
          <Col>
          <Link to={sortedStatus.length > 0 ? SEQUENCER_EXPLORER + sortedStatus[sortedStatus.length - 1].address:""}>
            <Card
              title={sortedStatus.length > 0 ?"Epoch: " + sortedStatus[sortedStatus.length - 1].id:""}
              loading={sortedStatus.length > 0 ?null:"true"}
              key={"next-seq"}
              hoverable
              style={{ width: 270, height: 450}}
              bordered={false}
              cover={
                <img
                  alt={sortedStatus.length > 0 ?sortedStatus[sortedStatus.length - 1].address:""}
                  src={sortedStatus.length > 0 ?sortedStatus[sortedStatus.length - 1].avatar:""}
                  style={{ height: 270, objectFit: "cover" }}
                />
              }
              className="criclebox"
            >
              <Meta 
              loading={sortedStatus.length > 0 ?null:"true"}
              title={sortedStatus.length > 0 ?sortedStatus[sortedStatus.length - 1].name:""} 
              description={sortedStatus.length > 0 ?truncateText(sortedStatus[sortedStatus.length - 1].desc,80):""}/>
            </Card>
            </Link>
          </Col>
        </Row>

        <div className="space30"></div>

        <Row gutter={[32, 32]}>
          <Banner 
           images={sortedStatus.length > 4 
            ? sortedStatus.slice(0, sortedStatus.length - 3).map((item) => ({
                id: item.id, 
                image: item.avatar,
                name: item.name,
                desc: item.desc,
                address: item.address
              }))
            : []
          }  
          />
          {/* <Banner images={images} /> */}
        </Row>
        </div>

        <div className="space30"></div>

        <SignStatus />
    </>
  );
};

export default Dashboard;