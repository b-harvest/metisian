import {Card, Col} from 'antd';
import { Link } from "react-router-dom";


import { SEQUENCER_EXPLORER, truncateText } from '../../pages/Dashboard'

const { Meta } = Card;

  
const Banner = ({ images, speed = 100000 }) => {
    // console.log(images)
    return (
      <div className="inner">
        <div className="wrapper">
          <section style={{ "--speed": `${speed}ms` }}>
            {images.map(({ id, image, name, desc = "", address = "" }) => (
            <Link to={SEQUENCER_EXPLORER + address}>
                <Col style={{ height: 150 }}>
                    <Card
                    loading={name ?null:"true"}
                    key={id}
                    hoverable
                    style={{ width: 150, height: 200 }} // Adjusted width and height
                    bordered={true}
                    cover={<img alt={id} src={image} style={{ width: "150px", height: "100px", objectFit: "cover" }} />}
                    className="criclebox"
                    >
                    <Meta title={name} description={truncateText(desc, 100)} style={{ fontSize: 8 }} /> {/* Adjusted font size */}
                    </Card>
                </Col>
            </Link>
                
            ))}
          </section>
          <section style={{ "--speed": `${speed}ms` }}>
            {images.map(({ id, image, name, desc = "", address = "" }) => (
            <Link to={SEQUENCER_EXPLORER + address}>
                <Col style={{ height: 150 }}>
                    <Card
                    loading={name ?null:"true"}
                    key={id}
                    hoverable
                    style={{ width: 150, height: 200 }} // Adjusted width and height
                    bordered={true}
                    cover={<img alt={id} src={image} style={{ width: "150px", height: "100px", objectFit: "cover" }} />}
                    className="criclebox"
                    >
                    <Meta title={name} description={truncateText(desc, 100)} style={{ fontSize: 8 }} /> {/* Adjusted font size */}
                    </Card>
                </Col>
            </Link>
                
            ))}
          </section>
          <section style={{ "--speed": `${speed}ms` }}>
            {images.map(({ id, image, name, desc = "", address = "" }) => (
            <Link to={SEQUENCER_EXPLORER + address}>
                <Col style={{ height: 150 }}>
                    <Card
                    loading={name ?null:"true"}
                    key={id}
                    hoverable
                    style={{ width: 150, height: 200 }} // Adjusted width and height
                    bordered={true}
                    cover={<img alt={id} src={image} style={{ width: "150px", height: "100px", objectFit: "cover" }} />}
                    className="criclebox"
                    >
                    <Meta title={name} description={truncateText(desc, 100)} style={{ fontSize: 8 }} /> {/* Adjusted font size */}
                    </Card>
                </Col>
            </Link>
                
            ))}
          </section>
        </div>
      </div>
    );
  };


export { Banner };