import React from 'react';
import { Container, Row, Col } from 'react-bootstrap';

function MinimalDashboard() {
  return (
    <Container fluid>
      <Row>
        <Col>
          <h1>Minimal Dashboard</h1>
          <p>This is a minimal version of the dashboard with only basic components.</p>
        </Col>
      </Row>
    </Container>
  );
}

export default MinimalDashboard;