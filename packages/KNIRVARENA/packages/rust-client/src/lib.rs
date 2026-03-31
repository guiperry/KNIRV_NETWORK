//! KNIRVANA Rust Game Client
//! 
//! A high-performance, cross-platform game client for the KNIRVANA ecosystem
//! built with Rust and Bevy engine.
//! 
//! ## Features
//! 
//! - Cross-platform support (Desktop, Android, iOS)
//! - Mobile optimizations with adaptive quality
//! - Blockchain integration with XION and NRN tokens
//! - Real-time networking and multiplayer support
//! - 3D graphics with physics simulation
//! - Modular ECS architecture
//! 
//! ## Quick Start
//! 
//! ```rust,no_run
//! use bevy::prelude::*;
//! use knirvana_game_client::*;
//! 
//! fn main() {
//!     App::new()
//!         .add_plugins(DefaultPlugins)
//!         .add_plugins(GameEnginePlugin)
//!         .add_plugins(NetworkingPlugin)
//!         .add_plugins(MobilePlugin)
//!         .run();
//! }
//! ```

pub mod game_engine;
pub mod components;
pub mod systems;
pub mod resources;
pub mod networking;
pub mod mobile;
pub mod knirvana_systems;
pub mod knirvana_ui;
pub mod nrn_economics;

// Re-export commonly used types
pub use game_engine::*;
pub use components::*;
pub use systems::*;
pub use resources::*;
pub use networking::*;
pub use mobile::*;
pub use knirvana_systems::*;
pub use knirvana_ui::*;
pub use nrn_economics::*;

// Re-export Bevy for convenience
pub use bevy::prelude::*;
pub use bevy_rapier3d::prelude::*;
