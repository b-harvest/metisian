import {} from 'react-router-dom';
import "antd/dist/antd.css";
import "./assets/styles/main.css";
import "./assets/styles/responsive.css";
import { Layout } from 'antd';

import { AppContextProvider } from '@/context/appContext';
import { StatusContextProvider } from '@/context/statusContext';
import { Suspense, lazy } from 'react';
import { BrowserRouter } from 'react-router-dom';
import AppRouter from './router/AppRouter';
import PageLoader from '@/components/PageLoader';

function App() {
  const { Content } = Layout;
  return (
    <BrowserRouter>
      <AppContextProvider>
        <StatusContextProvider>
          <Suspense fallback={<PageLoader />}>
            <div className="App">
              <Layout>
                <Content
                  style={{
                    margin: '40px auto 30px',
                    overflow: 'initial',
                    width: '100%',
                    padding: '0 50px',
                    maxWidth: 1400,
                  }}
                >
                  <AppRouter />
                </Content>
              </Layout>
            </div>
          </Suspense>
        </StatusContextProvider>
      </AppContextProvider>
    </BrowserRouter>
  );
}

export default App;
