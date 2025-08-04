use bevy::prelude::*;
use crate::components::*;
use crate::resources::*;

pub struct MobilePlugin;

impl Plugin for MobilePlugin {
    fn build(&self, app: &mut App) {
        app
            .init_resource::<MobileSettings>()
            .add_systems(Startup, setup_mobile_optimizations)
            .add_systems(Update, (
                mobile_performance_monitor,
                adaptive_quality_system,
                battery_optimization_system,
            ));
    }
}

fn setup_mobile_optimizations(
    mut commands: Commands,
    mut mobile_settings: ResMut<MobileSettings>,
) {
    #[cfg(feature = "mobile")]
    {
        info!("Setting up mobile optimizations");

        mobile_settings.touch_sensitivity = 1.0;
        mobile_settings.graphics_quality = GraphicsQuality::Medium;
        mobile_settings.battery_optimization = true;
        mobile_settings.reduced_effects = true;

        // Apply mobile-specific entity optimizations
        commands.spawn((
            MobileOptimized {
                lod_level: 2,
                simplified_physics: true,
                reduced_particles: true,
            },
            Name::new("Mobile Optimization Manager"),
        ));
    }

    #[cfg(not(feature = "mobile"))]
    {
        mobile_settings.graphics_quality = GraphicsQuality::High;
        mobile_settings.battery_optimization = false;
        mobile_settings.reduced_effects = false;
    }
}

fn mobile_performance_monitor(
    time: Res<Time>,
    mut mobile_settings: ResMut<MobileSettings>,
    mut metrics: ResMut<GameMetrics>,
) {
    #[cfg(feature = "mobile")]
    {
        metrics.frame_rate = 1.0 / time.delta_seconds();

        // Adaptive quality based on performance
        if metrics.frame_rate < 30.0 {
            match mobile_settings.graphics_quality {
                GraphicsQuality::High => {
                    mobile_settings.graphics_quality = GraphicsQuality::Medium;
                    info!("Reduced graphics quality to Medium due to low FPS");
                }
                GraphicsQuality::Medium => {
                    mobile_settings.graphics_quality = GraphicsQuality::Low;
                    info!("Reduced graphics quality to Low due to low FPS");
                }
                _ => {}
            }
        } else if metrics.frame_rate > 55.0 {
            match mobile_settings.graphics_quality {
                GraphicsQuality::Low => {
                    mobile_settings.graphics_quality = GraphicsQuality::Medium;
                    info!("Increased graphics quality to Medium");
                }
                GraphicsQuality::Medium => {
                    mobile_settings.graphics_quality = GraphicsQuality::High;
                    info!("Increased graphics quality to High");
                }
                _ => {}
            }
        }
    }
}

fn adaptive_quality_system(
    mobile_settings: Res<MobileSettings>,
    mut mobile_optimized_query: Query<&mut MobileOptimized>,
) {
    #[cfg(feature = "mobile")]
    {
        if mobile_settings.is_changed() {
            for mut optimized in mobile_optimized_query.iter_mut() {
                match mobile_settings.graphics_quality {
                    GraphicsQuality::Low => {
                        optimized.lod_level = 3;
                        optimized.simplified_physics = true;
                        optimized.reduced_particles = true;
                    }
                    GraphicsQuality::Medium => {
                        optimized.lod_level = 2;
                        optimized.simplified_physics = true;
                        optimized.reduced_particles = true;
                    }
                    GraphicsQuality::High => {
                        optimized.lod_level = 1;
                        optimized.simplified_physics = false;
                        optimized.reduced_particles = false;
                    }
                    GraphicsQuality::Ultra => {
                        optimized.lod_level = 0;
                        optimized.simplified_physics = false;
                        optimized.reduced_particles = false;
                    }
                }
            }
        }
    }
}

fn battery_optimization_system(
    mobile_settings: Res<MobileSettings>,
    time: Res<Time>,
) {
    #[cfg(feature = "mobile")]
    {
        if mobile_settings.battery_optimization {
            // Reduce update frequency for non-critical systems
            let frame_skip = (time.elapsed_seconds() * 60.0) as u32 % 2;

            if frame_skip == 0 {
                // Skip some updates to save battery
                return;
            }
        }
    }
}

// Mobile-specific input handling
pub fn handle_mobile_gestures(
    touches: Res<Touches>,
    mut player_query: Query<&mut Transform, With<PlayerAvatar>>,
    mobile_settings: Res<MobileSettings>,
) {
    #[cfg(feature = "mobile")]
    {
        // Handle pinch-to-zoom
        if touches.iter().count() == 2 {
            let touches: Vec<_> = touches.iter().collect();
            if touches.len() == 2 {
                let touch1 = touches[0].position();
                let touch2 = touches[1].position();
                let distance = touch1.distance(touch2);

                // Implement zoom logic based on pinch distance
                info!("Pinch gesture detected, distance: {}", distance);
            }
        }

        // Handle swipe gestures
        for touch in touches.iter() {
            if let Some(previous_position) = touches.get_pressed(touch.id()) {
                let current_position = touch.position();
                let delta = current_position - previous_position;

                if delta.length() > 100.0 * mobile_settings.touch_sensitivity {
                    info!("Swipe gesture detected: {:?}", delta);
                }
            }
        }
    }
}

// Platform-specific initialization
#[cfg(target_os = "android")]
pub fn android_init() {
    use android_logger::{Config, FilterBuilder};

    android_logger::init_once(
        Config::default()
            .with_min_level(log::Level::Info)
            .with_tag("KNIRVANA")
            .with_filter(FilterBuilder::new().parse("debug,hello::crate=trace").build())
    );

    info!("Android platform initialized");
}

#[cfg(target_os = "ios")]
pub fn ios_init() {
    info!("iOS platform initialized");
}
