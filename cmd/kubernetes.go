/*
Copyright © 2020 BizFly Cloud

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/bizflycloud/bizflyctl/formatter"
	"github.com/bizflycloud/gobizfly"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	kubernetesClusterHeader    = []string{"ID", "Name", "VPCNetworkID", "WorkerPoolsCount", "ClusterStatus", "Tags"}
	kubernetesWorkerPoolHeader = []string{"ID", "Name", "Version", "Flavor", "VolumeSize", "VolumeType", "EnabledAutoScaling", "MinSize", "MaxSize"}
	nodeID                     string
	clusterName                string
	clusterVersion             string
	autoUpgrade                bool
	vpcNetworkID               string
	enableCloud                bool
	tags                       []string
	workerPools                []string
	desiredSize                int
	enableAutoScaling          bool
	minSize                    int
	maxSize                    int
	outputKubeConfigFilePath   string
	inputConfigFile            string
)

var kubernetesCmd = &cobra.Command{
	Use:   "kubernetes",
	Short: "Bizfly Kubernetes Engine Interaction",
	Long:  "Bizfly Kubernetes Engine Action: List, Create, Delete, Get",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("kubernetes engine called")
	},
}
var clusterList = &cobra.Command{
	Use:   "list",
	Short: "List your Kubernetes cluster",
	Long:  "List your Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		clusters, err := client.KubernetesEngine.List(ctx, &gobizfly.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}
		var data [][]string
		for _, cluster := range clusters {
			data = append(data, []string{cluster.UID, cluster.Name, cluster.VPCNetworkID, strconv.Itoa(cluster.WorkerPoolsCount),
				cluster.ClusterStatus, strings.Join(cluster.Tags, ", ")})
		}
		formatter.Output(kubernetesClusterHeader, data)
	}}

var clusterCreate = &cobra.Command{
	Use:   "create",
	Short: "Create Kubernetes cluster with worker pool",
	Long: `Create Kubernetes cluster with worker pool using file or flags (Sample config file in example)
- Using flag example: ./bizflycloud kubernetes create --name test_cli --version 5f7d3a91d857155ad4993a32 --vpc-network-id 145bed1f-a7f7-4f88-ab3d-ce2fc95a4e71 -tag abc -tag xyz --worker-pool name=testworkerpool,flavor=nix.3c_6g,profile_type=premium,volume_type=PREMIUM-HDD1,volume_size=40,availability_zone=HN1,desired_size=1,min_size=1,max_size=10
- Using config file example: ./bizflycloud kubernetes create --config-file create_cluster.json`,
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		if inputConfigFile != "" {
			fileBytes, err := ioutil.ReadFile(inputConfigFile)
			if err != nil {
				log.Fatal(err)
			}
			var ccr *gobizfly.ClusterCreateRequest
			if err := json.Unmarshal(fileBytes, &ccr); err != nil {
				log.Fatal(err)
			}
			cluster, err := client.KubernetesEngine.Create(ctx, ccr)
			if err != nil {
				log.Fatal(err)
			}
			var data [][]string
			data = append(data, []string{cluster.UID, cluster.Name, cluster.VPCNetworkID, strconv.Itoa(cluster.WorkerPoolsCount),
				cluster.ClusterStatus, strings.Join(cluster.Tags, ", ")})
			formatter.Output(kubernetesClusterHeader, data)
		} else {
			workerPoolObjs := make([]gobizfly.WorkerPool, 0)
			for _, pool := range workerPools {
				workerPoolObjs = append(workerPoolObjs, parseWorkerPool(pool))
			}
			cluster, err := client.KubernetesEngine.Create(ctx, &gobizfly.ClusterCreateRequest{
				Name:         clusterName,
				Version:      clusterVersion,
				AutoUpgrade:  autoUpgrade,
				VPCNetworkID: vpcNetworkID,
				EnableCloud:  enableCloud,
				WorkerPools:  workerPoolObjs,
				Tags:         tags,
			})
			if err != nil {
				log.Fatal(err)
			}
			var data [][]string
			data = append(data, []string{cluster.UID, cluster.Name, cluster.VPCNetworkID, strconv.Itoa(cluster.WorkerPoolsCount),
				cluster.ClusterStatus, strings.Join(cluster.Tags, ", ")})
			formatter.Output(kubernetesClusterHeader, data)
		}
	},
}

var clusterGet = &cobra.Command{
	Use:   "get",
	Short: "Get Kubernetes cluster with worker pool",
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		cluster, err := client.KubernetesEngine.Get(ctx, args[0])
		if err != nil {
			log.Fatal(err)
		}
		var data [][]string
		data = append(data, []string{cluster.UID, cluster.Name, cluster.VPCNetworkID, strconv.Itoa(cluster.WorkerPoolsCount),
			cluster.ClusterStatus, strings.Join(cluster.Tags, ", ")})
		formatter.Output(kubernetesClusterHeader, data)
	},
}

var clusterDelete = &cobra.Command{
	Use:   "delete",
	Short: "Delete Kubernetes cluster with worker pool",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			log.Fatal("Invalid arguments")
		}
		client, ctx := getApiClient(cmd)
		err := client.KubernetesEngine.Delete(ctx, args[0])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Delete the Kubernetes cluster successfully")
	},
}

var addWorkerPool = &cobra.Command{
	Use:   "add-workerpools",
	Short: "Add worker pool into cluster",
	Long: `Add Kubernetes worker pool using file or flags (Sample config file in example)
- Using flag example: ./bizflycloud kubernetes add-workerpool xfbxsws38dcs8o94 --worker-pool name=testworkerpool,flavor=nix.3c_6g,profile_type=premium,volume_type=PREMIUM-HDD1,volume_size=40,availability_zone=HN1,desired_size=1,min_size=1,max_size=10
- Using config file example: ./bizflycloud kubernetes add-workerpool 55viixy9ma6yaiwu --config-file add_pools.json`,
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		if inputConfigFile != "" {
			fileBytes, err := ioutil.ReadFile(inputConfigFile)
			if err != nil {
				log.Fatal(err)
			}
			var awpr *gobizfly.AddWorkerPoolsRequest
			if err := json.Unmarshal(fileBytes, &awpr); err != nil {
				log.Fatal(err)
			}
			workerPools, err := client.KubernetesEngine.AddWorkerPools(ctx, args[0], awpr)
			if err != nil {
				log.Fatal(err)
			}
			var data [][]string
			for _, workerPool := range workerPools {
				data = append(data, []string{workerPool.UID, workerPool.Name, workerPool.Version, workerPool.Flavor,
					strconv.Itoa(workerPool.VolumeSize), workerPool.VolumeType, strconv.FormatBool(workerPool.EnableAutoScaling),
					strconv.Itoa(workerPool.MinSize), strconv.Itoa(workerPool.MaxSize)})
			}
			formatter.Output(kubernetesWorkerPoolHeader, data)

		} else {
			workerPoolObjs := make([]gobizfly.WorkerPool, 0)
			for _, pool := range workerPools {
				workerPoolObjs = append(workerPoolObjs, parseWorkerPool(pool))
			}
			workerPools, err := client.KubernetesEngine.AddWorkerPools(ctx, args[0], &gobizfly.AddWorkerPoolsRequest{
				WorkerPools: workerPoolObjs,
			})
			if err != nil {
				log.Fatal(err)
			}
			var data [][]string
			for _, workerPool := range workerPools {
				data = append(data, []string{workerPool.UID, workerPool.Name, workerPool.Version, workerPool.Flavor,
					strconv.Itoa(workerPool.VolumeSize), workerPool.VolumeType, strconv.FormatBool(workerPool.EnableAutoScaling),
					strconv.Itoa(workerPool.MinSize), strconv.Itoa(workerPool.MaxSize)})

				formatter.Output(kubernetesWorkerPoolHeader, data)
			}
		}
	},
}

var recycleNode = &cobra.Command{
	Use:   "recycle-node",
	Short: "Recycle Node",
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		err := client.KubernetesEngine.RecycleNode(ctx, args[0], args[1], args[2])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Recycling node successfully")
	},
}

var deleteWorkerPool = &cobra.Command{
	Use:   "delete-workerpool",
	Short: "Delete worker pool",
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		err := client.KubernetesEngine.DeleteClusterWorkerPool(ctx, args[0], args[1])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Delete worker pool successfully")
	},
}

var getWorkerPool = &cobra.Command{
	Use:   "get-workerpool",
	Short: "Get worker pool",
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		workerPool, err := client.KubernetesEngine.GetClusterWorkerPool(ctx, args[0], args[1])
		if err != nil {
			log.Fatal(err)
		}
		var data [][]string
		data = append(data, []string{workerPool.UID, workerPool.Name, workerPool.Version, workerPool.Flavor,
			strconv.Itoa(workerPool.VolumeSize), workerPool.VolumeType, strconv.FormatBool(workerPool.EnableAutoScaling),
			strconv.Itoa(workerPool.MinSize), strconv.Itoa(workerPool.MaxSize)})
		formatter.Output(kubernetesWorkerPoolHeader, data)
	},
}

var updateWorkerPool = &cobra.Command{
	Use:   "update-workerpool",
	Short: "Update worker pool",
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		uwr := &gobizfly.UpdateWorkerPoolRequest{
			DesiredSize:       desiredSize,
			EnableAutoScaling: enableAutoScaling,
			MinSize:           minSize,
			MaxSize:           maxSize,
		}
		err := client.KubernetesEngine.UpdateClusterWorkerPool(ctx, args[0], args[1], uwr)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Update worker pool successfully")
	},
}

var deleteWorkerPoolNode = &cobra.Command{
	Use:   "delete-node",
	Short: "Delete node",
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		err := client.KubernetesEngine.DeleteClusterWorkerPoolNode(ctx, args[0], args[1], args[2])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Delete worker pool node successfully")
	},
}

var getKubeConfig = &cobra.Command{
	Use:   "get-kubeconfig",
	Short: "Get kubeconfig",
	Run: func(cmd *cobra.Command, args []string) {
		client, ctx := getApiClient(cmd)
		resp, err := client.KubernetesEngine.GetKubeConfig(ctx, args[0])
		if err != nil {
			log.Fatal(err)
		}
		fileName := fmt.Sprintf("%s.kubeconfig", args[0])
		file, _ := os.Create(filepath.Join(outputKubeConfigFilePath, fileName))
		_, err = file.WriteString(resp)

		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Get kubernetes config successfully")
	},
}

func isIntField(key string) bool {
	for _, field := range []string{"volume_size", "desired_size", "min_size", "max_size"} {
		if field == key {
			return true
		}
	}
	return false
}
func parseWorkerPool(workerPoolStr string) gobizfly.WorkerPool {
	pairs := strings.Split(workerPoolStr, ",")
	strFieldMap := make(map[string]string)
	intFieldMap := make(map[string]int)
	isEnableAutoScaling := false
	r := regexp.MustCompile("(.*)=(.*)")
	for _, pair := range pairs {
		subStrs := r.FindStringSubmatch(pair)
		if len(subStrs) == 0 {
			log.Fatal("Invalid worker pool input")
		}
		fmt.Println(subStrs, len(subStrs))
		key, value := subStrs[1], subStrs[2]
		if key == "enable_autoscaling" {
			b, _ := strconv.ParseBool(value)
			isEnableAutoScaling = b
		}
		if isIntField(key) {
			i, _ := strconv.Atoi(value)
			intFieldMap[key] = i
		} else {
			strFieldMap[key] = value
		}
	}
	workerPool := gobizfly.WorkerPool{
		Name:              strFieldMap["name"],
		Version:           strFieldMap["version"],
		Flavor:            strFieldMap["flavor"],
		ProfileType:       strFieldMap["profile_type"],
		VolumeType:        strFieldMap["volume_type"],
		VolumeSize:        intFieldMap["volume_size"],
		AvailabilityZone:  strFieldMap["availability_zone"],
		DesiredSize:       intFieldMap["desired_size"],
		EnableAutoScaling: isEnableAutoScaling,
		MinSize:           intFieldMap["min_size"],
		MaxSize:           intFieldMap["max_size"],
	}
	fmt.Printf("%v+", workerPool)
	return workerPool
}

func init() {
	rootCmd.AddCommand(kubernetesCmd)
	kubernetesCmd.AddCommand(clusterList)
	kubernetesCmd.AddCommand(clusterDelete)
	kubernetesCmd.AddCommand(clusterGet)
	kubernetesCmd.AddCommand(recycleNode)
	kubernetesCmd.AddCommand(deleteWorkerPool)
	kubernetesCmd.AddCommand(getWorkerPool)
	kubernetesCmd.AddCommand(deleteWorkerPoolNode)

	kccq := clusterCreate.PersistentFlags()
	kccq.StringVar(&inputConfigFile, "config-file", "", "Input config file")
	kccq.StringVar(&clusterName, "name", "", "Name of cluster")
	kccq.StringVar(&clusterVersion, "version", "", "Version of cluster")
	kccq.BoolVar(&autoUpgrade, "auto-upgrade", false, "Auto Upgrade")
	kccq.StringVar(&vpcNetworkID, "vpc-network-id", "", "VPC Network ID")
	kccq.BoolVar(&enableCloud, "enable-cloud", false, "Enable Cloud")
	kccq.StringArrayVar(&tags, "tag", []string{}, "Tags of cluster")
	kccq.StringArrayVar(&workerPools, "worker-pool", []string{}, "Worker pools")
	kubernetesCmd.AddCommand(clusterCreate)

	awp := addWorkerPool.PersistentFlags()
	awp.StringVar(&inputConfigFile, "config-file", "", "Input config file")
	awp.StringArrayVar(&workerPools, "worker-pool", []string{}, "Worker pools")
	kubernetesCmd.AddCommand(addWorkerPool)

	uwp := updateWorkerPool.PersistentFlags()
	uwp.IntVar(&desiredSize, "desired-size", 0, "Desired size")
	uwp.BoolVar(&enableAutoScaling, "autoscaling", false, "Enable Auto scaling")
	uwp.IntVar(&minSize, "min-size", 0, "Min size")
	uwp.IntVar(&maxSize, "max-size", 0, "Max size")
	kubernetesCmd.AddCommand(updateWorkerPool)

	getKubeConfig.PersistentFlags().StringVar(&outputKubeConfigFilePath, "output-path", ".", "Output path")
	kubernetesCmd.AddCommand(getKubeConfig)
}