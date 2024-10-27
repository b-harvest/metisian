import {React, useContext, useEffect, useState } from 'react';

import {
  Card,
  Col,
  Row,
  Spin,
  Avatar
} from "antd";

import 'react-horizontal-scrolling-menu/dist/styles.css';
import { Link } from "react-router-dom";
import '../assets/styles/app.css'
import mining from "../assets/images/mining.gif"

import { Banner } from '../components/Banner'

import SignStatus from "../components/SignStatus";
import { useSeqStatus } from '../context/statusContext';

const {Meta} = Card

const MAINNET_SEQUENCER_RESOURCE = "https://raw.githubusercontent.com/MetisProtocol/metis-sequencer-resources/main/sequencers/1088";

const SEPOLIA_SEQUENCER_RESOURCE_BASE = "https://raw.githubusercontent.com/MetisProtocol/metis-sequencer-resources/main/sequencers/59902";
const ALL_ENDPOINT = "/all.json";

export const SEQUENCER_EXPLORER = "https://sequencer.metis.io/#/sequencers/"


export const truncateText = (text, maxLength) => {
  if (text == null) {
    return ""
  }
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength) + '...';
};


function Dashboard() {
  const { statusData } = useSeqStatus();
  const [seqResourceData, setSeqResourceData] = useState([]);

  const [sortedStatus, setSortedStatus] = useState([]);


  // useEffect(() => console.log(statusData), [statusData])

  useEffect(() => {
    loadSeqResource()
    // console.log(seqResourceData); // debug only
  }, [])

  const loadSeqResource = async () => {
    const seqResource = await fetch(MAINNET_SEQUENCER_RESOURCE + ALL_ENDPOINT, {
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
  
    if (statusData && Array.isArray(statusData.Status)) {
      statusData.Status.forEach((item) => {
        const matchedResource = seqResourceData.find(resourceItem =>
          item.address.toLowerCase() === resourceItem.seq_addr.toLowerCase()
        );
    
        if (matchedResource && Array.isArray(item.epochs)) {
          item.epochs.forEach((epoch) => {
            newEpochStat.push({
              address: item.address,
              avatar: matchedResource.avatar.replace("{BASEDIR}", MAINNET_SEQUENCER_RESOURCE),
              id: epoch,
              name: item.name,
              desc: matchedResource.desc,
              isProducing: item.is_producing
            });
          });
        }
      });
    }
    
  
    const statuses = [...newEpochStat].sort((a, b) => a.id - b.id);
    const fixedStatuses = [];
    fixedStatuses.push(...statuses)

    const lastSeq = statuses[statuses.length-1]
    if (statuses.length > 0 && lastSeq.isProducing) {
      fixedStatuses.push({
        id: lastSeq.id + 1,
        address: "",
        name: "Not Selected",
        desc: "waiting for new sequencer...",
        notSelected: true
      });
    }
    

    // console.log(fixedStatuses)
    setSortedStatus(fixedStatuses);

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
              title={
                sortedStatus.length > 2 ? (
                  <div style={{ display: 'flex', alignItems: 'center' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', position: 'absolute', top: '14px', right: '20px' }}>
                    <span style={{ fontSize: '14px', fontWeight: '500', color: '#02d6d6' }}>Completed</span>
                      <div
                        style={{
                          width: '12px',
                          height: '12px',
                          backgroundColor: '#02d6d6',
                          borderRadius: '50%',
                        }}
                      ></div>
                    </div>
                    {"Epoch: " + sortedStatus[sortedStatus.length - 3].id}
                  </div>
                ) : (
                  ""
                )
              }
              loading={sortedStatus.length > 2 ?null:"true"}
              key={"previous-seq"}
              hoverable
              style={{ width: 270, height: 450 }}
              bordered={false}
              cover={
                <img
                  alt={sortedStatus.length > 2 ?sortedStatus[sortedStatus.length - 3].address:""}
                  src={sortedStatus.length > 2 ?sortedStatus[sortedStatus.length - 3].avatar:null}
                  style={{ height: 270, objectFit: "cover", borderRadius: "50%"  }}
                /> 
              }
              className="criclebox"
            >
              <Meta 
              loading={sortedStatus.length > 2 ?null:"true"}
              title={sortedStatus.length > 2 ?sortedStatus[sortedStatus.length - 3].name:""} 
              description={sortedStatus.length > 2 ?truncateText(sortedStatus[sortedStatus.length - 3].desc,50):""}/>
            </Card>
            </Link>
          </Col>
          <Col>
            <Link to={sortedStatus.length > 1 ? SEQUENCER_EXPLORER + sortedStatus[sortedStatus.length - 2].address:""}>
            <Card
              title={
                sortedStatus.length > 1 ? (
                  <div style={{ display: 'flex', alignItems: 'center' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', position: 'absolute', top: '14px', right: '20px' }}>
                    <span style={{ fontSize: '14px', fontWeight: '500', color: '#00EA5E' }}>Producing</span>
                      <div
                        style={{
                          width: '12px',
                          height: '12px',
                          backgroundColor: '#00EA5E',
                          borderRadius: '50%',
                        }}
                      ></div>
                    </div>
                    {"Epoch: " + sortedStatus[sortedStatus.length - 2].id}
                  </div>
                ) : (
                  ""
                )
              }
              loading={sortedStatus.length > 1 ?null:"true"}
              key={"current-seq"}
              hoverable 
              style={{ 
                width: 300, 
                height: 500,
              }} 
              bordered={false} 
              cover={
                <img 
                  alt={sortedStatus.length > 1 ?sortedStatus[sortedStatus.length - 2].address:""} 
                  src={sortedStatus.length > 1 ?sortedStatus[sortedStatus.length - 2].avatar:null} 
                  style={{ height: 300, objectFit: "cover", borderRadius: "50%"  }}
                />
              }
              className="criclebox"
            >
              <Meta 
              loading={sortedStatus.length > 1 ?null:"true"}
              title={sortedStatus.length > 1 ?sortedStatus[sortedStatus.length - 2].name:""} 
              description={sortedStatus.length > 1 ?truncateText(sortedStatus[sortedStatus.length - 2].desc,100):""}
              />
            </Card>
            </Link>
          </Col>
          <Col>
            <Link to={sortedStatus.length > 0 ? SEQUENCER_EXPLORER + sortedStatus[sortedStatus.length - 1].address:""}>
            <Card
              title={
                sortedStatus.length > 0 ? (
                  <div style={{ display: 'flex', alignItems: 'center' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', position: 'absolute', top: '14px', right: '20px' }}>
                    <span style={{ fontSize: '14px', fontWeight: '500', color: '#00EA5E' }}>Next Period</span>
                      <div
                        style={{
                          width: '12px',
                          height: '12px',
                          backgroundColor: '#00EA5E',
                          borderRadius: '50%',
                        }}
                      ></div>
                    </div>
                    {"Epoch: " + sortedStatus[sortedStatus.length - 1].id}
                  </div>
                ) : (
                  ""
                )
              }
              loading={sortedStatus.length > 0 ?null:"true"}
              key={"next-seq"}
              hoverable
              style={{ width: 270, height: 450}}
              bordered={false}
              cover={ 
                <img
                  alt={sortedStatus.length > 0 ?sortedStatus[sortedStatus.length - 1].address:""}
                  src={sortedStatus.length > 0 ?sortedStatus[sortedStatus.length - 1].avatar:null}
                  style={{ height: 270, objectFit: "cover", borderRadius: "50%" }}
                />
              }
              className="criclebox"
            >
              <Meta 
              loading={sortedStatus.length > 0 && !sortedStatus[sortedStatus.length - 1].notSelected ?null:"true"}
              title={sortedStatus.length > 0 ?sortedStatus[sortedStatus.length - 1].name:""} 
              description={sortedStatus.length > 0 ?truncateText(sortedStatus[sortedStatus.length - 1].desc,50):""}/>
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