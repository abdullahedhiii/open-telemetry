import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import Home from '../pages/Home'

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Home/>} />
        <Route path="/stocks" element={<></>} />
        <Route path="/coins" element={<></>} />
        <Route path="/stockInfo" element={<></>} />
        <Route path="/coinIfnfo" element={<></>} />
      </Routes>
    </Router>
  )
}

export default App