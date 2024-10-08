import {Card, Col} from 'antd';
import { Link } from "react-router-dom";


import { SEQUENCER_EXPLORER, truncateText } from '../../pages/Dashboard'

const { Meta } = Card;

  
const Banner = ({ images, speed = 50000 }) => {

    // Helper function to shuffle an array
    const shuffleArray = (array) => {
    for (let i = array.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [array[i], array[j]] = [array[j], array[i]]; // Swap elements
    }
    return array;
  };
  
  // Function to get doubled shuffled images
  const getDoubledShuffledImages = (images) => {
    // Shuffle images and remove duplicates based on their 'name' property
    let shuffled = shuffleArray(images).filter((imageObj, index, self) =>
      index === self.findIndex((t) => t.name === imageObj.name)
    );
    
    // Concatenate the shuffled array with itself
    const doubledShuffledImages = [...shuffled, ...shuffled];
    
    return doubledShuffledImages;
  };

  const doubledShuffledImages = getDoubledShuffledImages(images);

    // console.log(images)
    return (
      <div className="inner">
        <div className="wrapper">
          <section style={{ "--speed": `${speed}ms` }}>
            {doubledShuffledImages.map(({ id, image, name, desc = "", address = "" }) => (
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
                    <Meta title={name} description={truncateText(desc, 80)} style={{ fontSize: 8 }} /> {/* Adjusted font size */}
                    </Card>
                </Col>
            </Link>
                
            ))}
          </section>
          <section style={{ "--speed": `${speed}ms` }}>
            {doubledShuffledImages.map(({ id, image, name, desc = "", address = "" }) => (
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
                    <Meta title={name} description={truncateText(desc, 80)} style={{ fontSize: 8 }} /> {/* Adjusted font size */}
                    </Card>
                </Col>
            </Link>
                
            ))}
          </section>
          <section style={{ "--speed": `${speed}ms` }}>
            {doubledShuffledImages.map(({ id, image, name, desc = "", address = "" }) => (
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
                    <Meta title={name} description={truncateText(desc, 80)} style={{ fontSize: 8 }} /> {/* Adjusted font size */}
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