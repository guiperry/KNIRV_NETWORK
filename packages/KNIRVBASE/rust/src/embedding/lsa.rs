use serde::{Deserialize, Serialize};
use std::f64::consts::SQRT_2;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LsaReducer {
    components: Vec<Vec<f64>>,
    target_dim: usize,
    source_dim: usize,
    mean: Vec<f64>,
}

impl LsaReducer {
    pub fn new(target_dim: usize) -> Self {
        Self {
            components: Vec::new(),
            target_dim,
            source_dim: 0,
            mean: Vec::new(),
        }
    }

    pub fn fit(&mut self, vectors: &[Vec<f64>]) -> Result<(), String> {
        if vectors.is_empty() {
            return Err("no vectors provided".to_string());
        }

        self.source_dim = vectors[0].len();
        let num_vectors = vectors.len();

        self.mean = vec![0.0; self.source_dim];
        for vec in vectors {
            for (i, val) in vec.iter().enumerate() {
                self.mean[i] += val;
            }
        }
        for val in self.mean.iter_mut() {
            *val /= num_vectors as f64;
        }

        let mut centered_vectors: Vec<Vec<f64>> = Vec::new();
        for vec in vectors {
            let mut centered = Vec::with_capacity(self.source_dim);
            for i in 0..vec.len() {
                centered.push(vec[i] - self.mean[i]);
            }
            centered_vectors.push(centered);
        }

        self.components.clear();
        for k in 0..self.target_dim.min(self.source_dim) {
            let component = self.extract_component(&centered_vectors, k);
            self.components.push(component);

            for centered in centered_vectors.iter_mut() {
                let projection = dot_product(centered, self.components[k].as_slice());
                for j in 0..centered.len() {
                    centered[j] -= projection * self.components[k][j];
                }
            }
        }

        Ok(())
    }

    fn extract_component(&self, vectors: &[Vec<f64>], _component_idx: usize) -> Vec<f64> {
        if vectors.is_empty() || self.source_dim == 0 {
            return vec![0.0; self.source_dim];
        }

        let mut component = vec![1.0 / SQRT_2; self.source_dim];

        let max_iterations = 100;
        for _ in 0..max_iterations {
            let mut new_component = vec![0.0; self.source_dim];
            for vec in vectors {
                let proj = dot_product(vec, &component);
                for j in 0..new_component.len() {
                    new_component[j] += proj * vec[j];
                }
            }

            let norm = vector_norm(&new_component);
            if norm > 0.0 {
                for j in 0..new_component.len() {
                    new_component[j] /= norm;
                }
            }

            let mut diff = 0.0;
            for j in 0..component.len() {
                diff += (new_component[j] - component[j]).abs();
            }
            component = new_component;

            if diff < 1e-6 {
                break;
            }
        }

        component
    }

    pub fn transform(&self, vector: &[f64]) -> Result<Vec<f64>, String> {
        if vector.len() != self.source_dim {
            return Err(format!(
                "vector dimension mismatch: expected {}, got {}",
                self.source_dim,
                vector.len()
            ));
        }

        let mut centered = Vec::with_capacity(vector.len());
        for i in 0..vector.len() {
            centered.push(vector[i] - self.mean[i]);
        }

        let mut reduced = vec![0.0; self.target_dim];
        for (i, component) in self.components.iter().enumerate() {
            if i < self.target_dim {
                reduced[i] = dot_product(&centered, component);
            }
        }

        Ok(reduced)
    }

    pub fn transform_batch(&self, vectors: &[Vec<f64>]) -> Result<Vec<Vec<f64>>, String> {
        let mut reduced = Vec::new();
        for vec in vectors {
            reduced.push(self.transform(vec)?);
        }
        Ok(reduced)
    }

    pub fn fit_transform(&mut self, vectors: &[Vec<f64>]) -> Result<Vec<Vec<f64>>, String> {
        self.fit(vectors)?;
        self.transform_batch(vectors)
    }

    pub fn target_dimension(&self) -> usize {
        self.target_dim
    }

    pub fn source_dimension(&self) -> usize {
        self.source_dim
    }
}

fn dot_product(a: &[f64], b: &[f64]) -> f64 {
    if a.len() != b.len() {
        return 0.0;
    }

    a.iter().zip(b.iter()).map(|(x, y)| x * y).sum()
}

fn vector_norm(v: &[f64]) -> f64 {
    v.iter().map(|x| x * x).sum::<f64>().sqrt()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_lsa_transform() {
        let mut reducer = LsaReducer::new(2);

        let vectors = vec![
            vec![1.0, 2.0, 3.0],
            vec![2.0, 4.0, 6.0],
            vec![1.5, 3.0, 4.5],
        ];

        reducer.fit(&vectors).unwrap();

        let result = reducer.transform(&[1.0, 2.0, 3.0]).unwrap();
        assert_eq!(result.len(), 2);
    }

    #[test]
    fn test_lsa_dimension_mismatch() {
        let reducer = LsaReducer::new(2);

        let result = reducer.transform(&[1.0, 2.0, 3.0, 4.0, 5.0]);
        assert!(result.is_err());
    }
}
