import React from 'react';
import { Container, Row, Col, Card } from 'react-bootstrap';

function SimpleDashboard() {
  const darkBlue = '#000c2e';
  const brightBlue = '#007bff';
  
  return (
    <Container fluid style={{ minHeight: '100vh', backgroundColor: darkBlue, color: 'white' }}>
      <Row className="p-4">
        <Col>
          <h1 style={{ color: brightBlue }}>Blockchain Dashboard</h1>
          <p>This is a simplified version of the dashboard.</p>
        </Col>
      </Row>
      
      <Row className="p-4">
        <Col md={6}>
          <Card bg="dark" text="white" className="mb-4">
            <Card.Body>
              <Card.Title style={{ color: brightBlue }}>Dashboard Card</Card.Title>
              <Card.Text>
                This is a simple card component from React Bootstrap.
              </Card.Text>
            </Card.Body>
          </Card>
        </Col>
        
        <Col md={6}>
          <Card bg="dark" text="white">
            <Card.Body>
              <Card.Title style={{ color: brightBlue }}>Analytics</Card.Title>
              <Card.Text>
                This card contains analytics information.
              </Card.Text>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </Container>
  );
}

export default SimpleDashboard;