import React from 'react';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Card from 'react-bootstrap/Card';
import Navbar from 'react-bootstrap/Navbar';
import Nav from 'react-bootstrap/Nav';
import Form from 'react-bootstrap/Form';
// Import icons from react-icons/bs
import {
  BsSearch, BsBell, BsGear, BsPersonCircle,
  BsGrid1X2, BsWallet, BsWallets,
  BsDiagram3, BsHeart, BsStack, BsGraphUp
} from 'react-icons/bs';

function BlockchainDashboard() {
  const darkBlue = '#000c2e';
  const brightBlue = '#007bff';

  const glassyBackgroundStyle = {
    backgroundColor: 'rgba(255, 255, 255, 0.05)',
    backdropFilter: 'blur(8px)',
    borderRadius: '15px',
    border: '1px solid rgba(255, 255, 255, 0.1)',
    boxShadow: '0 8px 32px 0 rgba(0, 0, 0, 0.1)',
  };

  const cardGlassyStyle = {
    ...glassyBackgroundStyle,
    backgroundColor: 'rgba(255, 255, 255, 0.08)',
    border: '1px solid rgba(255, 255, 255, 0.15)',
    boxShadow: '0 4px 16px 0 rgba(0, 0, 0, 0.08)',
  };

  const activeNavLinkStyle = {
    ...glassyBackgroundStyle,
    backgroundColor: brightBlue,
    color: 'white',
    fontWeight: 'bold',
  };

  const navLinkStyle = {
    ...glassyBackgroundStyle,
    color: 'rgba(255, 255, 255, 0.8)',
  };

  return (
    <Container fluid style={{ minHeight: '100vh', backgroundColor: darkBlue, color: 'white' }}>
      <Row>
        <Col md={2} className="p-4" style={{ backgroundColor: 'rgba(0, 0, 0, 0.6)' }}>
          {/* Sidebar */}
          <Navbar bg="transparent" variant="dark" className="flex-column">
            <Navbar.Brand href="#" className="mb-4" style={{ fontWeight: 'bold', color: brightBlue }}>Blockchain</Navbar.Brand>
            <Nav className="flex-column">
              <Nav.Item>
                <Nav.Link href="#" active style={activeNavLinkStyle}>
                  <BsGrid1X2 className="me-2" />
                  Menu
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link href="#" style={navLinkStyle}>
                  <BsWallet className="me-2" />
                  Wallet
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link href="#" style={navLinkStyle}>
                  <BsWallets className="me-2" />
                  Wallets
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link href="#" style={navLinkStyle}>
                  <BsDiagram3 className="me-2" />
                  Networks
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link href="#" style={navLinkStyle}>
                  <BsHeart className="me-2" />
                  Donations
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link href="#" style={navLinkStyle}>
                  <BsStack className="me-2" />
                  Farm/Drops
                </Nav.Link>
              </Nav.Item>
              <Nav.Item>
                <Nav.Link href="#" style={navLinkStyle}>
                  <BsGraphUp className="me-2" />
                  Prediction
                </Nav.Link>
              </Nav.Item>
            </Nav>
          </Navbar>
        </Col>

        <Col md={10} className="p-4">
          {/* Top Navigation */}
          <Navbar bg="transparent" expand="lg" className="mb-4" style={glassyBackgroundStyle}>
            <Container fluid>
              <Navbar.Brand href="#" style={{ fontWeight: 'bold', color: brightBlue }}>Blockchain</Navbar.Brand>
              <Navbar.Toggle aria-controls="basic-navbar-nav" />
              <Navbar.Collapse id="basic-navbar-nav">
                <Nav className="me-auto"></Nav>
                <Form className="d-flex me-2">
                  <Form.Control
                    type="search"
                    placeholder="Search blockchain"
                    className="me-2"
                    aria-label="Search"
                    style={glassyBackgroundStyle}
                    variant="dark"
                  />
                </Form>
                <Nav className="ms-auto">
                  <Nav.Item>
                    <Nav.Link href="#" style={navLinkStyle}><BsSearch color={brightBlue} /></Nav.Link>
                  </Nav.Item>
                  <Nav.Item>
                    <Nav.Link href="#" style={navLinkStyle}><BsBell color={brightBlue} /></Nav.Link>
                  </Nav.Item>
                  <Nav.Item>
                    <Nav.Link href="#" style={navLinkStyle}><BsGear color={brightBlue} /></Nav.Link>
                  </Nav.Item>
                  <Nav.Item>
                    <Nav.Link href="#" style={navLinkStyle}><BsPersonCircle color={brightBlue} /></Nav.Link>
                  </Nav.Item>
                </Nav>
              </Navbar.Collapse>
            </Container>
          </Navbar>

          {/* Main Content */}
          <Row className="mb-4">
            <Col>
              <h2 style={{ fontWeight: 'bold', color: brightBlue }}>Blockchain</h2>
              <p className="text-muted" style={{ color: 'rgba(255, 255, 255, 0.7)' }}>Navigated files</p>
            </Col>
          </Row>

          <Row className="mb-4">
            <Col md={3}>
              <Card style={cardGlassyStyle} className="text-center text-white">
                <Card.Body>
                  <Card.Title style={{ fontWeight: 'bold', color: brightBlue }}>Refu</Card.Title>
                  <Card.Text>
                    <h3 style={{ fontWeight: 'bold' }}>15.556</h3>
                    <small className="text-success">+2.3%</small>
                  </Card.Text>
                </Card.Body>
              </Card>
            </Col>
            <Col md={3}>
              <Card style={cardGlassyStyle} className="text-center text-white">
                <Card.Body>
                  <Card.Title style={{ fontWeight: 'bold', color: brightBlue }}>Dratic</Card.Title>
                  <Card.Text>
                    <h3 style={{ fontWeight: 'bold' }}>255.0%</h3>
                    <small className="text-danger">-1.1%</small>
                  </Card.Text>
                </Card.Body>
              </Card>
            </Col>
            <Col md={3}>
              <Card style={cardGlassyStyle} className="text-center text-white">
                <Card.Body>
                  <Card.Title style={{ fontWeight: 'bold', color: brightBlue }}>Hapler</Card.Title>
                  <Card.Text>
                    <h3 style={{ fontWeight: 'bold' }}>997</h3>
                    <small className="text-success">+0.8%</small>
                  </Card.Text>
                </Card.Body>
              </Card>
            </Col>
            <Col md={3}>
              <Card style={cardGlassyStyle} className="text-center text-white">
                <Card.Body>
                  <Card.Title style={{ fontWeight: 'bold', color: brightBlue }}>Coetic</Card.Title>
                  <Card.Text>
                    <h3 style={{ fontWeight: 'bold' }}>461.53</h3>
                    <small className="text-success">+0.5%</small>
                  </Card.Text>
                </Card.Body>
              </Card>
            </Col>
          </Row>

          <Row>
            <Col>
              <Card style={cardGlassyStyle} className="text-white">
                <Card.Body>
                  <Card.Title style={{ fontWeight: 'bold', color: brightBlue }}>Analytics Overview</Card.Title>
                  {/* Placeholder for Chart */}
                  <div style={{ height: '200px', backgroundColor: 'rgba(224, 247, 250, 0.1)', border: '1px dashed rgba(178, 235, 242, 0.3)', borderRadius: '10px', display: 'flex', justifyContent: 'center', alignItems: 'center', color: brightBlue }}>
                    <span>Chart Area Placeholder</span>
                  </div>
                  <Row className="mt-3">
                    <Col md={6}>
                      <Card style={cardGlassyStyle} className="text-white">
                        <Card.Body>
                          <Card.Title style={{ fontWeight: 'bold', color: brightBlue }}>Navigation</Card.Title>
                          <Card.Text>
                            <h2 style={{ fontWeight: 'bold' }}>15,0.5K</h2>
                          </Card.Text>
                        </Card.Body>
                      </Card>
                    </Col>
                    <Col md={6}></Col>
                  </Row>
                </Card.Body>
              </Card>
            </Col>
          </Row>
        </Col>
      </Row>
    </Container>
  );
}

export default BlockchainDashboard;

