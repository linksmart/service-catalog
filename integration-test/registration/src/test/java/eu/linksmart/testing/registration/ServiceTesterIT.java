package eu.linksmart.testing.registration;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.swagger.client.ApiClient;
import io.swagger.client.api.ScApi;
import io.swagger.client.model.APIIndex;
import io.swagger.client.model.Service;
import io.swagger.client.model.ServiceDocs;
import org.junit.Test;

import java.io.File;
import java.math.BigDecimal;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

public class ServiceTesterIT {
    static final String
            BASE_URL = "http://localhost:8082",
            DEFAULT_FILE_NAME = "test/dummy.json";

    static final String
            BASE_URL_PATH = "base_url",
            FILENAME = "filename";
    @Test
    public void registration(){

        System.out.println("Start registration dummy integration test");
        ApiClient client = new ApiClient();
        ObjectMapper mapper = new ObjectMapper();

        System.out.println("SC URL: "+System.getenv().getOrDefault(BASE_URL_PATH, BASE_URL));
        client.setBasePath(System.getenv().getOrDefault(BASE_URL_PATH, BASE_URL));
        ScApi api = new ScApi(client);

        System.out.println("Verification registration file : "+System.getenv().getOrDefault(FILENAME, DEFAULT_FILE_NAME));
        String id = UUID.randomUUID().toString(), file = System.getenv().getOrDefault(FILENAME, DEFAULT_FILE_NAME);

        try{
            Service service = mapper.readValue(new File(file), Service.class);

            System.out.println("Registering service");
            Service service2= api.idPut(id,service);

            Service service1=api.idGet(id);

            assertTrue("Ids must be equal", id.equals(service1.getId()));
            comp(service,service1);

            assertTrue("Ids must be equal", id.equals(service2.getId()));
            comp(service1,service2);

            APIIndex index =  api.rootGet(new BigDecimal(1),new BigDecimal(100));
            assertTrue("It must contain 1 service", index.getTotal().equals(1));

            api.idDelete(id);

            index =  api.rootGet(new BigDecimal(1),new BigDecimal(100));
            assertTrue("It must be empty", index.getTotal().equals(0));

        }catch (Exception e){
            e.printStackTrace();
            fail();
        }

        System.out.println("Registration Dummy integration test finished");

    }
    private void comp(Service s1, Service s2){

        assertTrue("Name must be equal", s1.getName().equals(s2.getName()));
        assertTrue("Description must be equal", s1.getDescription().equals(s2.getDescription()));
        assertTrue("Docs must be equal", s1.getDocs().equals(s2.getDocs()));
        assertTrue("Apis must be equal", s1.getApis().equals(s2.getApis()));
        assertTrue("Meta must be equal", s1.getMeta().equals(s2.getMeta()));

    }
}
