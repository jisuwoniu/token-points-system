const hre = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();
  
  console.log("Deploying contracts with account:", deployer.address);
  console.log("Account balance:", (await ethers.provider.getBalance(deployer.address)).toString());

  const TokenPoints = await ethers.getContractFactory("TokenPoints");
  
  const name = "Token Points";
  const symbol = "TPTS";
  const initialSupply = ethers.parseEther("1000000"); // 1,000,000 tokens

  const token = await TokenPoints.deploy(name, symbol, initialSupply);
  await token.waitForDeployment();

  const tokenAddress = await token.getAddress();
  
  console.log("TokenPoints deployed to:", tokenAddress);
  console.log("Network:", hre.network.name);
  console.log("Chain ID:", hre.network.config.chainId);

  // Wait for a few block confirmations
  console.log("Waiting for block confirmations...");
  await token.deploymentTransaction().wait(5);

  // Verify contract on Etherscan
  if (hre.network.name !== "hardhat" && process.env.ETHERSCAN_API_KEY) {
    console.log("Verifying contract on Etherscan...");
    try {
      await hre.run("verify:verify", {
        address: tokenAddress,
        constructorArguments: [name, symbol, initialSupply],
      });
      console.log("Contract verified successfully");
    } catch (error) {
      console.log("Verification failed:", error.message);
    }
  }

  // Save deployment info
  const deploymentInfo = {
    network: hre.network.name,
    chainId: hre.network.config.chainId,
    contractAddress: tokenAddress,
    deployer: deployer.address,
    deploymentTime: new Date().toISOString(),
    transactionHash: token.deploymentTransaction().hash
  };

  const fs = require("fs");
  const deploymentFile = `deployments/${hre.network.name}.json`;
  
  if (!fs.existsSync("deployments")) {
    fs.mkdirSync("deployments");
  }
  
  fs.writeFileSync(deploymentFile, JSON.stringify(deploymentInfo, null, 2));
  console.log("Deployment info saved to:", deploymentFile);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
