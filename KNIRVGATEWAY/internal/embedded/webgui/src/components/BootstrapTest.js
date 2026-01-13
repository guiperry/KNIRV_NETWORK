import React from 'react';
// Import each component individually to test
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Card from 'react-bootstrap/Card';
import Button from 'react-bootstrap/Button';

function BootstrapTest() {
  return (
    <Container>
      <Row className="my-4">
        <Col>
          <h1>Bootstrap Test</h1>
          <p>Testing individual react-bootstrap component imports</p>
        </Col>
      </Row>
      <Row>
        <Col md={6}>
          <Card>
            <Card.Body>
              <Card.Title>Test Card</Card.Title>
              <Card.Text>
                This is a test card using individual imports.
              </Card.Text>
              <Button variant="primary">Test Button</Button>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </Container>
  );
}

export default BootstrapTest;