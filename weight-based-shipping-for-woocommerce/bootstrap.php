<?php

use Wbs\Plugin;


call_user_func(function() {

    $pluginEntryFile = wp_normalize_path(__DIR__.'/plugin.php');

    register_activation_hook($pluginEntryFile, function() {

        $otherPluginVersions = [];
        foreach (get_option('active_plugins') as $pluginFile) {
            $data = get_file_data(WP_PLUGIN_DIR.'/'.$pluginFile, ['PluginFamilyId' => 'Plugin Family Id']);
            $id = @$data['PluginFamilyId'];
            if ($id === 'dangoodman/wc-weight-based-shipping') {
                $otherPluginVersions[] = $pluginFile;
            }
        }

        if ($otherPluginVersions) {
            deactivate_plugins($otherPluginVersions);
        }
    });

    if (!class_exists(Plugin::class, false)) {
        require_once(__DIR__."/server/vendor/autoload.php");
        Plugin::setupOnce($pluginEntryFile);
    }
});