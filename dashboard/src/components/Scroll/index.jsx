import { ScrollMenu } from 'react-horizontal-scrolling-menu';
import { LeftArrow, RightArrow } from '../Arrow';
import {Card, Col} from 'antd';
import styled from 'styled-components';


const { Meta } = Card;

export function Scrolls(items) {

  return (
    <Container>
        <ScrollMenu 
        LeftArrow={LeftArrow} RightArrow={RightArrow} >
            {items.items.map(
                ({ itemId }) => {
                    return (
                      <Col 
                      xs={24}
                      sm={24}
                      md={12}
                      lg={6}
                      xl={6}>
                        <Card
                            key={itemId}
                            hoverable 
                            style={{ width: 240 }} 
                            bordered={false} 
                            cover={<img alt="example" src="https://os.alipayobjects.com/rmsportal/QBnOOoLaAfKPirc.png" />}
                            className="criclebox "
                        >
                        <Meta title="Europe Street beat" description="www.instagram.com"/>
                        </Card>
                      </Col>

                    );
                },
            )}
        </ScrollMenu>
    </Container>
  );
}
const Container = styled.div`
  overflow: hidden;
  .react-horizontal-scrolling-menu--scroll-container::-webkit-scrollbar {
    display: none;
  }
  .react-horizontal-scrolling-menu--scroll-container {
    -ms-overflow-style: none; /* IE and Edge */
    scrollbar-width: none; /* Firefox */
  }
`;
