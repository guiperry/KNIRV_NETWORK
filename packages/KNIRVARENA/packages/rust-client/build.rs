fn main() {
    // Platform-specific build configuration

    #[cfg(target_os = "android")]
    {
        println!("cargo:rustc-link-lib=log");
        println!("cargo:rustc-link-lib=android");
    }

    #[cfg(target_os = "ios")]
    {
        println!("cargo:rustc-link-lib=framework=Foundation");
        println!("cargo:rustc-link-lib=framework=UIKit");
    }
}
