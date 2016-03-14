// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import stateConfig from './daemonsetlist_stateconfig';
import logsMenuDirective from './logsmenu_directive';
import filtersModule from 'common/filters/filters_module';
import componentsModule from 'common/components/components_module';
import daemonSetCardDirective from './daemonsetcard_directive';
import daemonSetCardMenuDirective from './daemonsetcardmenu_directive';
import daemonSetDetailModule from 'daemonsetdetail/daemonsetdetail_module';
import daemonSetListContainer from './daemonsetlistcontainer_directive';

/**
 * Angular module for the DaemonSet list view.
 *
 * The view shows DaemonSets running in the cluster and allows to manage them.
 */
export default angular
    .module(
        'kubernetesDashboard.daemonSetList',
        [
          'ngMaterial',
          'ngResource',
          'ui.router',
          daemonSetDetailModule.name,
          filtersModule.name,
          componentsModule.name,
        ])
    .config(stateConfig)
    .directive('logsMenu', logsMenuDirective)
    .directive('kdDaemonSetListContainer', daemonSetListContainer)
    .directive('kdDaemonSetCard', daemonSetCardDirective)
    .directive('kdDaemonSetCardMenu', daemonSetCardMenuDirective);
