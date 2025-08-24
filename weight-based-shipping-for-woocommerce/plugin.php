<?php
/**
 * Plugin Name: Weight Based Shipping for WooCommerce
 * Plugin URI: https://wordpress.org/plugins/weight-based-shipping-for-woocommerce/
 * Description: Simple yet flexible shipping method for WooCommerce.
 * Version: 6.9.0
 * Author: weightbasedshipping.com
 * Author URI: https://weightbasedshipping.com
 * Requires PHP: 7.2
 * Requires at least: 4.6
 * Tested up to: 6.8
 * WC requires at least: 5.0
 * WC tested up to: 10.1
 */

if (!class_exists('WbsVendors\Dgm\WpPluginBootstrapGuard\Guard', false)) {
    require_once(__DIR__.'/server/vendor/dangoodman/wp-plugin-bootstrap-guard/Guard.php');
}
WbsVendors\Dgm\WpPluginBootstrapGuard\Guard::checkPrerequisitesAndBootstrap(__FILE__, __DIR__.'/bootstrap.php');