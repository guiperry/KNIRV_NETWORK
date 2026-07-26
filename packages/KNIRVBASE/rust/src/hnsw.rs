use rand::Rng;
use serde::{Deserialize, Serialize};
use std::cmp::Ordering;
use std::collections::{BinaryHeap, HashMap};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HNSWNode {
    pub id: Uuid,
    pub vector: Vec<f32>,
    pub connections: HashMap<i32, Vec<Uuid>>,
    pub level: i32,
}

#[derive(Debug, Clone)]
pub struct HNSWIndex {
    dimension: usize,
    m: usize,
    _mmax: usize,
    mmax0: usize,
    ef_construction: usize,
    ef: usize,
    _ml: f64,
    nodes: HashMap<Uuid, HNSWNode>,
    entry_point: Option<Uuid>,
}

#[derive(Debug)]
pub struct HNSWError {
    pub msg: String,
}

impl HNSWError {
    pub fn new(msg: &str) -> Self {
        Self {
            msg: msg.to_string(),
        }
    }
}

impl std::fmt::Display for HNSWError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.msg)
    }
}

impl std::error::Error for HNSWError {}

#[derive(Clone)]
struct DistanceItem {
    id: Uuid,
    distance: f64,
}

impl PartialEq for DistanceItem {
    fn eq(&self, other: &Self) -> bool {
        self.distance == other.distance
    }
}

impl Eq for DistanceItem {}

impl PartialOrd for DistanceItem {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for DistanceItem {
    fn cmp(&self, other: &Self) -> Ordering {
        self.distance
            .partial_cmp(&other.distance)
            .unwrap_or(Ordering::Equal)
    }
}

impl HNSWIndex {
    pub fn new(dimension: usize, m: usize, ef_construction: usize) -> Self {
        Self {
            dimension,
            m,
            _mmax: m,
            mmax0: m * 2,
            ef_construction,
            ef: ef_construction,
            _ml: 1.0 / (2.0_f64.ln()),
            nodes: HashMap::new(),
            entry_point: None,
        }
    }

    pub fn set_ef(&mut self, ef: usize) {
        self.ef = ef;
    }

    pub fn add(&mut self, id: Uuid, vector: Vec<f32>) -> Result<(), HNSWError> {
        if vector.len() != self.dimension {
            return Err(HNSWError::new("dimension mismatch"));
        }

        let level = self.random_level();

        let mut node = HNSWNode {
            id,
            vector,
            connections: HashMap::new(),
            level,
        };

        for l in 0..=level {
            node.connections.insert(l, Vec::new());
        }

        if self.entry_point.is_none() {
            self.entry_point = Some(id);
            self.nodes.insert(id, node);
            return Ok(());
        }

        let entry_point_id = self.entry_point.unwrap();
        let mut ep = vec![entry_point_id];

        if let Some(entry) = self.nodes.get(&entry_point_id) {
            for lc in (level + 1)..=entry.level {
                ep = self.search_layer(&node.vector, &ep, 1, lc);
            }
        }

        for lc in (0..=level).rev() {
            let m = if lc == 0 { self.mmax0 } else { self.m };
            let candidates = self.search_layer(&node.vector, &ep, self.ef_construction, lc);
            let neighbors = self.select_neighbors(&node.vector, &candidates, m);

            for &neighbor_id in &neighbors {
                self.connect(id, neighbor_id, lc);
                self.connect(neighbor_id, id, lc);

                if let Some(neighbor_node) = self.nodes.get(&neighbor_id) {
                    let conns = neighbor_node
                        .connections
                        .get(&lc)
                        .map(|c| c.len())
                        .unwrap_or(0);
                    if conns > m {
                        self.prune_connections(neighbor_id, lc, m);
                    }
                }
            }

            ep = candidates;
        }

        if level
            > self
                .nodes
                .get(&self.entry_point.unwrap())
                .map(|n| n.level)
                .unwrap_or(-1)
        {
            self.entry_point = Some(id);
        }

        self.nodes.insert(id, node);
        Ok(())
    }

    pub fn search(&self, query: &[f32], k: usize) -> Result<Vec<Uuid>, HNSWError> {
        if query.len() != self.dimension {
            return Err(HNSWError::new("dimension mismatch"));
        }

        let entry = match self.entry_point {
            Some(id) => id,
            None => return Ok(Vec::new()),
        };

        let mut ep = if self.nodes.len() <= k * 2 {
            self.nodes.keys().cloned().collect::<Vec<_>>()
        } else {
            vec![entry]
        };

        if let Some(entry_node) = self.nodes.get(&entry) {
            for lc in (1..=entry_node.level).rev() {
                ep = self.search_layer(query, &ep, 1, lc);
            }
        }

        ep = self.search_layer(query, &ep, self.ef.max(k), 0);

        let mut entries: Vec<(Uuid, f64)> = ep
            .iter()
            .filter_map(|id| {
                self.nodes.get(id).map(|node| {
                    let dist = euclidean_distance(query, &node.vector);
                    (*id, dist)
                })
            })
            .collect();

        entries.sort_by(|a, b| a.1.partial_cmp(&b.1).unwrap_or(Ordering::Equal));

        Ok(entries.into_iter().take(k).map(|(id, _)| id).collect())
    }

    fn search_layer(
        &self,
        query: &[f32],
        entry_points: &[Uuid],
        ef: usize,
        layer: i32,
    ) -> Vec<Uuid> {
        let mut candidates: BinaryHeap<DistanceItem> = BinaryHeap::new();
        let mut results: BinaryHeap<DistanceItem> = BinaryHeap::new();
        let mut visited: HashMap<Uuid, bool> = HashMap::new();

        for &ep in entry_points {
            if let Some(node) = self.nodes.get(&ep) {
                let dist = euclidean_distance(query, &node.vector);
                candidates.push(DistanceItem {
                    id: ep,
                    distance: dist,
                });
                results.push(DistanceItem {
                    id: ep,
                    distance: -dist,
                });
                visited.insert(ep, true);
            }
        }

        while let Some(current) = candidates.pop() {
            if results.len() >= ef
                && current.distance > -results.peek().map(|r| r.distance).unwrap_or(f64::MAX)
            {
                break;
            }

            if let Some(node) = self.nodes.get(&current.id) {
                if let Some(connections) = node.connections.get(&layer) {
                    for &neighbor_id in connections {
                        if !visited.contains_key(&neighbor_id) {
                            visited.insert(neighbor_id, true);

                            if let Some(neighbor) = self.nodes.get(&neighbor_id) {
                                let dist = euclidean_distance(query, &neighbor.vector);

                                if results.len() < ef
                                    || dist
                                        < -results.peek().map(|r| r.distance).unwrap_or(f64::MAX)
                                {
                                    candidates.push(DistanceItem {
                                        id: neighbor_id,
                                        distance: dist,
                                    });
                                    results.push(DistanceItem {
                                        id: neighbor_id,
                                        distance: -dist,
                                    });

                                    if results.len() > ef {
                                        results.pop();
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        let mut items: Vec<_> = results.into_vec();
        items.reverse();
        items.into_iter().map(|item| item.id).collect()
    }

    fn select_neighbors(&self, query: &[f32], candidates: &[Uuid], m: usize) -> Vec<Uuid> {
        if candidates.len() <= m {
            return candidates.to_vec();
        }

        let mut cands: Vec<(Uuid, f64)> = candidates
            .iter()
            .filter_map(|id| {
                self.nodes.get(id).map(|node| {
                    let dist = euclidean_distance(query, &node.vector);
                    (*id, dist)
                })
            })
            .collect();

        cands.sort_by(|a, b| a.1.partial_cmp(&b.1).unwrap_or(Ordering::Equal));

        cands.into_iter().take(m).map(|(id, _)| id).collect()
    }

    fn connect(&mut self, from: Uuid, to: Uuid, layer: i32) {
        if let Some(node) = self.nodes.get_mut(&from) {
            if let Some(connections) = node.connections.get_mut(&layer) {
                if !connections.contains(&to) {
                    connections.push(to);
                }
            }
        }
    }

    fn prune_connections(&mut self, node_id: Uuid, layer: i32, m: usize) {
        let node = match self.nodes.get(&node_id) {
            Some(n) => n,
            None => return,
        };

        let connections = match node.connections.get(&layer) {
            Some(c) => c.clone(),
            None => return,
        };

        if connections.len() <= m {
            return;
        }

        let mut distances: Vec<(Uuid, f64)> = connections
            .iter()
            .filter_map(|conn_id| {
                self.nodes.get(conn_id).map(|conn_node| {
                    let dist = euclidean_distance(&node.vector, &conn_node.vector);
                    (*conn_id, dist)
                })
            })
            .collect();

        distances.sort_by(|a, b| a.1.partial_cmp(&b.1).unwrap_or(Ordering::Equal));

        let new_connections: Vec<Uuid> = distances.into_iter().take(m).map(|(id, _)| id).collect();

        if let Some(node) = self.nodes.get_mut(&node_id) {
            node.connections.insert(layer, new_connections);
        }
    }

    fn random_level(&self) -> i32 {
        let mut rng = rand::thread_rng();
        let mut level = 0i32;
        while rng.gen::<f64>() < 0.5 && level < 16 {
            level += 1;
        }
        level
    }

    pub fn remove(&mut self, id: Uuid) -> Result<(), HNSWError> {
        let node = match self.nodes.remove(&id) {
            Some(n) => n,
            None => return Err(HNSWError::new("node not found")),
        };

        for layer in 0..=node.level {
            for &neighbor_id in node
                .connections
                .get(&layer)
                .map(|c| c.as_slice())
                .unwrap_or(&[])
            {
                if let Some(neighbor) = self.nodes.get_mut(&neighbor_id) {
                    if let Some(connections) = neighbor.connections.get_mut(&layer) {
                        connections.retain(|&conn| conn != id);
                    }
                }
            }
        }

        if self.entry_point == Some(id) {
            self.entry_point = self.find_new_entry_point();
        }

        Ok(())
    }

    fn find_new_entry_point(&self) -> Option<Uuid> {
        self.nodes
            .values()
            .max_by_key(|node| node.level)
            .map(|node| node.id)
    }

    pub fn size(&self) -> usize {
        self.nodes.len()
    }

    pub fn get_all_nodes(&self) -> Vec<Uuid> {
        self.nodes.keys().cloned().collect()
    }

    pub fn get_node(&self, id: &Uuid) -> Option<&HNSWNode> {
        self.nodes.get(id)
    }
}

fn euclidean_distance(v1: &[f32], v2: &[f32]) -> f64 {
    if v1.len() != v2.len() {
        return f64::MAX;
    }

    let mut sum = 0.0_f64;
    for i in 0..v1.len() {
        let diff = v1[i] as f64 - v2[i] as f64;
        sum += diff * diff;
    }

    sum.sqrt()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_hnsw_add_and_search() {
        let mut index = HNSWIndex::new(4, 16, 200);

        let id1 = Uuid::new_v4();
        let id2 = Uuid::new_v4();
        let id3 = Uuid::new_v4();

        index.add(id1, vec![1.0, 0.0, 0.0, 0.0]).unwrap();
        index.add(id2, vec![0.0, 1.0, 0.0, 0.0]).unwrap();
        index.add(id3, vec![0.9, 0.1, 0.0, 0.0]).unwrap();

        let results = index.search(&[1.0, 0.0, 0.0, 0.0], 2).unwrap();

        assert_eq!(results.len(), 2);
        assert_eq!(results[0], id1);
    }

    #[test]
    fn test_hnsw_remove() {
        let mut index = HNSWIndex::new(4, 16, 200);

        let id1 = Uuid::new_v4();
        let id2 = Uuid::new_v4();

        index.add(id1, vec![1.0, 0.0, 0.0, 0.0]).unwrap();
        index.add(id2, vec![0.0, 1.0, 0.0, 0.0]).unwrap();

        assert_eq!(index.size(), 2);

        index.remove(id1).unwrap();

        assert_eq!(index.size(), 1);
        assert!(index.get_node(&id1).is_none());
        assert!(index.get_node(&id2).is_some());
    }

    #[test]
    fn test_dimension_mismatch() {
        let mut index = HNSWIndex::new(4, 16, 200);

        let result = index.add(Uuid::new_v4(), vec![1.0, 0.0, 0.0]);
        assert!(result.is_err());
    }
}
