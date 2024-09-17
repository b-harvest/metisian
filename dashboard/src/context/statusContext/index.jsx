import React, { createContext, useState, useContext } from 'react';


const StatusContext = createContext();


export const StatusContextProvider = ({ children }) => {
  const [statusData, setStatusData] = useState([]);

  return (
    <StatusContext.Provider value={{ statusData, setStatusData }}>
      {children}
    </StatusContext.Provider>
  );
};


export const useSeqStatus = () => {
  return useContext(StatusContext);
};
